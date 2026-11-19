#!/usr/bin/env python3
"""Rebuild GBU project from 响应正本 §4.4 系统功能模块 + 报价明细功能点."""

from __future__ import annotations

import json
import re
import sqlite3
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

from docx import Document

BASE = "http://127.0.0.1:8080"
DB = "/Users/weining/workspace/ningpm/apps/api/data/ningpm.db"
DOC = "/Users/weining/workspace/dwq/docs/大湾区大学招生管理系统开发制作及后期维护服务采购项目（正本）.docx"
EMAIL = "admin@ningpm.local"
PASSWORD = "admin123"

# 4.4.x 模块名 → 报价表二级模块归并
MODULE_ORDER = [
    "4.4.1 综合评价",
    "4.4.2 生源管理",
    "4.4.3 招生活动",
    "4.4.4 在线缴费",
    "4.4.5 消息通知",
    "4.4.6 智能问答",
    "4.4.7 高考录取",
    "4.4.8 三端入口",
    "4.4.9 招办后台",
    "4.4.10 系统基础与安全",
]

QUOTE_TO_44 = {
    "综合评价": "4.4.1 综合评价",
    "生源管理": "4.4.2 生源管理",
    "招生活动": "4.4.3 招生活动",
    "在线缴费": "4.4.4 在线缴费",
    "消息通知": "4.4.5 消息通知",
    "智能问答": "4.4.6 智能问答",
    "高考录取": "4.4.7 高考录取",
    "考生与家长端": "4.4.8 三端入口",
    "中学与团体端": "4.4.8 三端入口",
    "展示与移动入口": "4.4.8 三端入口",
    "工作台与看板": "4.4.9 招办后台",
    "业务配置": "4.4.9 招办后台",
    "审核与考务": "4.4.9 招办后台",
    "生源与活动运营": "4.4.9 招办后台",
    "消息管理": "4.4.9 招办后台",
    "系统管理": "4.4.10 系统基础与安全",
    "安全与合规": "4.4.10 系统基础与安全",
}

# 开发迭代：按 4.2.2 上线依赖顺序
CYCLES = [
    ("W1 底座安全", "4.4.10 系统基础与安全：角色权限、字典集成、审计加密隔离", "active", "2026-11-18", "2026-11-23", ["4.4.10 系统基础与安全"]),
    ("W2 三端入口", "4.4.8 考生/中学/展示多端统一账号与报名入口", "planned", "2026-11-22", "2026-11-28", ["4.4.8 三端入口"]),
    ("W3 综评主线", "4.4.1 综合评价全链路 + 后台配置/审核考务承接", "planned", "2026-11-25", "2026-12-06", ["4.4.1 综合评价"]),
    ("W4 运营与共用", "4.4.2–4.4.6 生源/活动/缴费/消息/智能问答", "planned", "2026-11-29", "2026-12-10", [
        "4.4.2 生源管理", "4.4.3 招生活动", "4.4.4 在线缴费", "4.4.5 消息通知", "4.4.6 智能问答",
    ]),
    ("W5 录取与后台", "4.4.7 高考录取 + 4.4.9 招办工作台/统计/运营页", "planned", "2026-12-05", "2026-12-14", [
        "4.4.7 高考录取", "4.4.9 招办后台",
    ]),
]


def req(method: str, path: str, token: str | None = None, body: dict | None = None) -> Any:
    data = None if body is None else json.dumps(body, ensure_ascii=False).encode()
    headers = {"Content-Type": "application/json; charset=utf-8"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r) as resp:
            payload = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        raise SystemExit(f"{method} {path} -> {e.code}: {e.read().decode()}") from e
    if not payload.get("ok"):
        raise SystemExit(f"{method} {path} failed: {payload}")
    return payload.get("data")


def parse_quote() -> list[dict]:
    doc = Document(DOC)
    table = doc.tables[5]
    items = []
    for r in table.rows[1:]:
        cells = [c.text.strip().replace("\n", " ") for c in r.cells]
        if len(cells) < 5:
            continue
        seq, l1, l2, point, desc = cells[0], cells[1], cells[2], cells[3], cells[4]
        if not seq.isdigit():
            continue
        n = int(seq)
        # 4.4 系统开发：招生系统+三端+招办(1-34) + 安全合规落入 4.4.10(43-45)
        if not (1 <= n <= 34 or 43 <= n <= 45):
            continue
        reqs = []
        m = re.search(r"主要要求[:：](.*)$", desc)
        body = m.group(1) if m else ""
        for part in re.split(r"[①②③④⑤⑥⑦⑧⑨⑩]", body):
            part = part.strip(" ；;。")
            if part:
                reqs.append(part)
        explain = ""
        m2 = re.search(r"说明[:：](.*?)　主要要求", desc)
        if m2:
            explain = m2.group(1).strip()
        mod44 = QUOTE_TO_44.get(l2)
        if not mod44:
            continue
        items.append({
            "seq": n,
            "l2": l2,
            "point": point,
            "explain": explain,
            "reqs": reqs,
            "module44": mod44,
        })
    return items


def clear_gbu(project_id: str) -> None:
    con = sqlite3.connect(DB)
    cur = con.cursor()
    cur.execute("DELETE FROM comments WHERE issue_id IN (SELECT id FROM issues WHERE project_id=?)", (project_id,))
    try:
        cur.execute(
            "DELETE FROM issue_attachments WHERE issue_id IN (SELECT id FROM issues WHERE project_id=?)",
            (project_id,),
        )
    except sqlite3.OperationalError:
        pass
    cur.execute("DELETE FROM notifications WHERE project_id=?", (project_id,))
    cur.execute("DELETE FROM issues WHERE project_id=?", (project_id,))
    cur.execute("DELETE FROM cycles WHERE project_id=?", (project_id,))
    cur.execute("DELETE FROM modules WHERE project_id=?", (project_id,))
    con.commit()
    con.close()


def main() -> None:
    items = parse_quote()
    Path("/tmp/dwq-docs/quote-4.4-dev.json").write_text(
        json.dumps(items, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    print(f"parsed {len(items)} function points from §4.4 / quote")

    login = req("POST", "/api/auth/login", body={"email": EMAIL, "password": PASSWORD})
    token = login["token"]
    admin_id = login["user"]["id"]

    projects = req("GET", "/api/projects", token)
    gbu = next((p for p in projects if p.get("key") == "GBU"), None)
    if not gbu:
        gbu = req(
            "POST",
            "/api/projects",
            token,
            {
                "key": "GBU",
                "name": "大湾区大学招生管理系统",
                "description": "按响应正本 §4.4 系统功能模块（4.4.1–4.4.10）拆解的系统开发 backlog。",
            },
        )
    else:
        clear_gbu(gbu["id"])
        # refresh description via sqlite
        con = sqlite3.connect(DB)
        con.execute(
            "UPDATE projects SET description=? WHERE id=?",
            (
                "按响应正本 §4.4 系统功能模块（4.4.1–4.4.10）拆解的系统开发 backlog；功能点对齐报价明细。",
                gbu["id"],
            ),
        )
        con.commit()
        con.close()
    project_id = gbu["id"]
    print("project", project_id)

    modules: dict[str, str] = {}
    for name in MODULE_ORDER:
        md = req("POST", f"/api/projects/{project_id}/modules", token, {"name": name})
        modules[name] = md["id"]

    cycle_ids: dict[str, str] = {}
    module_cycle: dict[str, str] = {}
    for name, goal, status, start, end, mods in CYCLES:
        cy = req(
            "POST",
            f"/api/projects/{project_id}/cycles",
            token,
            {"name": name, "goal": goal, "status": status, "startDate": start, "endDate": end},
        )
        cycle_ids[name] = cy["id"]
        for m in mods:
            module_cycle[m] = cy["id"]

    def create_issue(**kwargs: Any) -> dict:
        return req("POST", "/api/issues", token, kwargs)

    epic_count = 0
    task_count = 0
    for it in items:
        mod = it["module44"]
        cycle_id = module_cycle.get(mod)
        # 招办后台里承接综评配置/审核的条目，挂到 W3 更贴开发主线
        if mod == "4.4.9 招办后台" and it["l2"] in ("业务配置", "审核与考务"):
            cycle_id = cycle_ids["W3 综评主线"]
        if mod == "4.4.9 招办后台" and it["l2"] in ("生源与活动运营", "消息管理"):
            cycle_id = cycle_ids["W4 运营与共用"]

        title = f"{it['point']}"
        if it["l2"] not in ("综合评价", "生源管理", "招生活动", "在线缴费", "消息通知", "智能问答", "高考录取"):
            title = f"{it['l2']} · {it['point']}"

        desc = it["explain"]
        if it["reqs"]:
            desc = (desc + "\n\n主要要求：\n" + "\n".join(f"①{r}" if i == 0 else f"{'①②③④⑤⑥⑦⑧⑨⑩'[i]}{r}" for i, r in enumerate(it["reqs"][:10]))).strip()

        epic = create_issue(
            projectId=project_id,
            type="epic",
            title=f"[4.4] {title}",
            description=desc or f"来自报价明细第 {it['seq']} 项 / §4.4",
            priority=1 if it["seq"] <= 6 else 2,
            moduleId=modules[mod],
            cycleId=cycle_id,
            assigneeId=admin_id,
            status="todo",
        )
        epic_count += 1

        for i, r in enumerate(it["reqs"]):
            create_issue(
                projectId=project_id,
                type="task",
                title=r[:120],
                description=f"父功能点：{title}（报价序号 {it['seq']}）",
                priority=2 if i < 2 else 3,
                moduleId=modules[mod],
                cycleId=cycle_id,
                parentId=epic["id"],
                assigneeId=admin_id,
                status="todo",
            )
            task_count += 1

    issues = req("GET", f"/api/issues?projectId={project_id}&rootsOnly=0", token)
    print(
        json.dumps(
            {
                "modules": len(modules),
                "cycles": len(cycle_ids),
                "epics": epic_count,
                "tasks": task_count,
                "totalIssues": len(issues),
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    main()
