#!/usr/bin/env python3
"""按 docs/投标功能完成度.xlsx 回填 GBU 史诗/子事项状态与说明。"""

from __future__ import annotations

import re
import sqlite3
from datetime import datetime, timezone

from openpyxl import load_workbook

DB = "/Users/weining/workspace/ningpm/apps/api/data/ningpm.db"
XLSX = "/Users/weining/workspace/dwq/docs/投标功能完成度.xlsx"
PROJECT_KEY = "GBU"


def epic_status(rate: float) -> str:
    if rate >= 0.8:
        return "done"
    if rate >= 0.6:
        return "review"
    if rate >= 0.4:
        return "doing"
    return "todo"


def match_key(l2: str, point: str) -> str:
    """与 seed-gbu-4.4 标题规则对齐（去掉 [4.4] 前缀后的主体）。"""
    core = ("综合评价", "生源管理", "招生活动", "在线缴费", "消息通知", "智能问答", "高考录取")
    if l2 in core:
        return point
    return f"{l2} · {point}"


def main() -> None:
    wb = load_workbook(XLSX, data_only=True)
    ws = wb["功能对照"]
    rows = []
    for r in ws.iter_rows(min_row=2, values_only=True):
        if not r or r[0] is None:
            continue
        try:
            seq = int(r[0])
        except (TypeError, ValueError):
            continue
        if seq > 48:
            continue
        l2 = str(r[2] or "").strip()
        point = str(r[3] or "").strip()
        rate = float(r[5] or 0)
        band = str(r[6] or "").strip()
        gap = str(r[8] or "").strip()
        rows.append(
            {
                "seq": seq,
                "l2": l2,
                "point": point,
                "rate": rate,
                "band": band,
                "gap": gap,
                "key": match_key(l2, point),
            }
        )

    con = sqlite3.connect(DB)
    con.row_factory = sqlite3.Row
    cur = con.cursor()
    proj = cur.execute("SELECT id FROM projects WHERE key = ?", (PROJECT_KEY,)).fetchone()
    if not proj:
        raise SystemExit("GBU project not found")
    pid = proj["id"]

    epics = cur.execute(
        """
        SELECT id, title, description FROM issues
        WHERE project_id = ? AND type = 'epic'
          AND (parent_id IS NULL OR parent_id = '')
        """,
        (pid,),
    ).fetchall()

    by_key: dict[str, sqlite3.Row] = {}
    for e in epics:
        title = e["title"]
        body = title[6:] if title.startswith("[4.4] ") else title
        by_key[body] = e

    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    matched = 0
    child_updates = 0

    for row in rows:
        epic = by_key.get(row["key"])
        if not epic:
            # 宽松：功能点名后缀匹配
            for k, e in by_key.items():
                if k.endswith(row["point"]) or row["point"] in k:
                    epic = e
                    break
        if not epic:
            print("MISS", row["seq"], row["key"])
            continue

        st = epic_status(row["rate"])
        note = (
            f"完成度 {row['rate']:.0%}（{row['band']}）｜报价序号 {row['seq']}\n"
            f"缺口：{row['gap'] or '—'}\n"
            f"来源：投标功能完成度.xlsx · 评估日 2026-11-17"
        )
        old_desc = (epic["description"] or "").strip()
        # 去掉旧回填块，保留原始主要要求
        old_desc = re.sub(
            r"(?s)\n*---\n完成度.*$",
            "",
            old_desc,
        ).strip()
        new_desc = f"{old_desc}\n\n---\n{note}".strip() if old_desc else note

        cur.execute(
            "UPDATE issues SET status = ?, description = ?, updated_at = ? WHERE id = ?",
            (st, new_desc, now, epic["id"]),
        )

        children = cur.execute(
            """
            SELECT id FROM issues
            WHERE parent_id = ?
            ORDER BY number, created_at
            """,
            (epic["id"],),
        ).fetchall()
        n = len(children)
        # ≥80% 视为已可用：子项全部完成；其余按完成度比例切分
        if n == 0:
            done_n = 0
        elif row["rate"] >= 0.8:
            done_n = n
        else:
            done_n = min(n, max(0, round(row["rate"] * n)))
        for i, ch in enumerate(children):
            if i < done_n:
                cst = "done"
            elif row["rate"] >= 0.6 and i == done_n:
                cst = "review"
            elif row["rate"] >= 0.4 and i == done_n:
                cst = "doing"
            else:
                cst = "todo"
            cur.execute(
                "UPDATE issues SET status = ?, updated_at = ? WHERE id = ?",
                (cst, now, ch["id"]),
            )
            child_updates += 1
        matched += 1
        print(
            f"OK #{row['seq']} {row['key'][:40]} rate={row['rate']:.0%} epic={st} children={done_n}/{n}"
        )

    con.commit()

    # 汇总
    stats = cur.execute(
        """
        SELECT
          COUNT(*) total,
          SUM(CASE WHEN parent_id IS NOT NULL AND parent_id != '' THEN 1 ELSE 0 END) children,
          SUM(CASE WHEN status='done' THEN 1 ELSE 0 END) done,
          SUM(CASE WHEN status='doing' THEN 1 ELSE 0 END) doing,
          SUM(CASE WHEN status='review' THEN 1 ELSE 0 END) review,
          SUM(CASE WHEN status='todo' THEN 1 ELSE 0 END) todo
        FROM issues WHERE project_id = ?
        """,
        (pid,),
    ).fetchone()
    con.close()
    print(
        f"matched_epics={matched}/{len(rows)} child_updates={child_updates} "
        f"total={stats['total']} children={stats['children']} "
        f"done={stats['done']} doing={stats['doing']} review={stats['review']} todo={stats['todo']}"
    )


if __name__ == "__main__":
    main()
