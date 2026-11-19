#!/usr/bin/env python3
"""Seed GBU 招生管理系统 project from procurement docs into local ningpm API."""

from __future__ import annotations

import json
import sqlite3
import sys
import urllib.error
import urllib.request
from typing import Any

BASE = "http://127.0.0.1:8080"
DB = "/Users/weining/workspace/ningpm/apps/api/data/ningpm.db"
EMAIL = "admin@ningpm.local"
PASSWORD = "admin123"


def req(method: str, path: str, token: str | None = None, body: dict | None = None) -> Any:
    data = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r) as resp:
            payload = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        err = e.read().decode()
        raise SystemExit(f"{method} {path} -> {e.code}: {err}") from e
    if not payload.get("ok"):
        raise SystemExit(f"{method} {path} failed: {payload}")
    return payload.get("data")


def main() -> None:
    login = req("POST", "/api/auth/login", body={"email": EMAIL, "password": PASSWORD})
    token = login["token"]
    admin_id = login["user"]["id"]
    print("login ok", admin_id)

    # Drop DEMO demo project rows if present (keep users).
    con = sqlite3.connect(DB)
    cur = con.cursor()
    row = cur.execute("SELECT id FROM projects WHERE key = 'DEMO'").fetchone()
    if row:
        pid = row[0]
        for tbl in (
            "comments",
            "issue_attachments",
            "notifications",
            "issues",
            "cycles",
            "modules",
            "project_members",
            "projects",
        ):
            try:
                if tbl == "projects":
                    cur.execute("DELETE FROM projects WHERE id = ?", (pid,))
                elif tbl == "project_members":
                    cur.execute("DELETE FROM project_members WHERE project_id = ?", (pid,))
                elif tbl == "modules":
                    cur.execute("DELETE FROM modules WHERE project_id = ?", (pid,))
                elif tbl == "cycles":
                    cur.execute("DELETE FROM cycles WHERE project_id = ?", (pid,))
                elif tbl == "issues":
                    cur.execute("DELETE FROM issues WHERE project_id = ?", (pid,))
                elif tbl == "notifications":
                    cur.execute("DELETE FROM notifications WHERE project_id = ?", (pid,))
                elif tbl == "comments":
                    cur.execute(
                        "DELETE FROM comments WHERE issue_id IN (SELECT id FROM issues WHERE project_id = ?)",
                        (pid,),
                    )
                elif tbl == "issue_attachments":
                    cur.execute(
                        "DELETE FROM issue_attachments WHERE issue_id IN (SELECT id FROM issues WHERE project_id = ?)",
                        (pid,),
                    )
            except sqlite3.OperationalError as e:
                print("skip", tbl, e)
        con.commit()
        print("removed DEMO", pid)
    con.close()

    project = req(
        "POST",
        "/api/projects",
        token,
        {
            "key": "GBU",
            "name": "大湾区大学招生管理系统",
            "description": (
                "采购项目 ZZ0260707｜预算 80 万｜建设期签约后 30 天上线试运行｜"
                "运维自验收起不少于 5 年。依据《采购文件》用户需求书 + 响应正本十模块方案拆解。"
            ),
        },
    )
    project_id = project["id"]
    print("project", project["key"], project_id)

    module_names = [
        "系统基础与安全",
        "门户与信息公开",
        "综合评价",
        "生源管理",
        "招生活动",
        "在线缴费",
        "消息通知",
        "智能问答",
        "高考录取",
        "三端入口",
        "招办后台与看板",
        "UI品牌视觉",
        "集成联调与运维交付",
    ]
    modules: dict[str, str] = {}
    for name in module_names:
        md = req("POST", f"/api/projects/{project_id}/modules", token, {"name": name})
        modules[name] = md["id"]
    print("modules", len(modules))

    # 以今天为 D1（本地计划基线；签约后可整体平移）
    cycles_spec = [
        (
            "S1 启动与需求",
            "D1–D4：启动会、调研、样板与对接清单确认",
            "active",
            "2026-11-18",
            "2026-11-21",
        ),
        (
            "S2 架构与设计",
            "D5–D9：环境开通、权限矩阵、字典基线、综评样板设计",
            "planned",
            "2026-11-22",
            "2026-11-26",
        ),
        (
            "S3 系统实现",
            "D10–D22：样板落地、通道联调、业务批量配置、三端与后台",
            "planned",
            "2026-11-27",
            "2026-12-09",
        ),
        (
            "S4 测试与切换",
            "D23–D27：功能/性能/安全/兼容测试、UAT、缺陷回归",
            "planned",
            "2026-12-10",
            "2026-12-14",
        ),
        (
            "S5 上线与支持",
            "D28–D30：生产切换、培训、巡检、验收材料",
            "planned",
            "2026-12-15",
            "2026-12-17",
        ),
    ]
    cycles: dict[str, str] = {}
    for name, goal, status, start, end in cycles_spec:
        cy = req(
            "POST",
            f"/api/projects/{project_id}/cycles",
            token,
            {
                "name": name,
                "goal": goal,
                "status": status,
                "startDate": start,
                "endDate": end,
            },
        )
        cycles[name.split()[0]] = cy["id"]
    print("cycles", list(cycles.keys()))

    def epic(title: str, module: str, desc: str, priority: int = 2) -> str:
        it = req(
            "POST",
            "/api/issues",
            token,
            {
                "projectId": project_id,
                "type": "epic",
                "title": title,
                "description": desc,
                "priority": priority,
                "moduleId": modules[module],
                "assigneeId": admin_id,
                "status": "todo",
            },
        )
        return it["id"]

    def task(
        title: str,
        module: str,
        cycle_key: str | None,
        parent: str | None = None,
        *,
        typ: str = "task",
        desc: str = "",
        priority: int = 3,
        status: str = "todo",
    ) -> str:
        body: dict[str, Any] = {
            "projectId": project_id,
            "type": typ,
            "title": title,
            "description": desc,
            "priority": priority,
            "moduleId": modules[module],
            "assigneeId": admin_id,
            "status": status,
        }
        if cycle_key:
            body["cycleId"] = cycles[cycle_key]
        if parent:
            body["parentId"] = parent
        it = req("POST", "/api/issues", token, body)
        return it["id"]

    # --- S1 启动与需求 ---
    e_kickoff = epic(
        "项目启动与需求确认",
        "集成联调与运维交付",
        "采购建设期 30 天阶段一：启动会、接口人、调研七清单、样板项目、对接资源到位。",
        1,
    )
    for title, desc, pri in [
        ("召开项目启动会并确认双方接口人", "输出《项目启动纪要》；明确校方业务/招办/信息中心/采购四类角色。", 1),
        ("完成业务调研七清单", "综评规则、活动类型、审核考务、消息触达、缴费财务、报表、外部依赖。", 1),
        ("选定 1–2 个综评/活动配置样板", "优先选政策最复杂场次，覆盖权重/多轮审核/补材。", 2),
        ("确认外部系统对接清单与到位计划", "统一身份、短信、邮件、微信、支付、教务、财务、门禁、物流等。", 1),
        ("界面原型走查并形成《需求确认纪要》", "在演示环境走通配置→审核→缴费→准考证；48h 内双方确认。", 2),
        ("VI/官网对照确认窗口安排", "品牌主色、字体、版式与学校官网一致（▲条款）。", 2),
    ]:
        task(title, "集成联调与运维交付", "S1", e_kickoff, desc=desc, priority=pri)

    # --- S2 架构与设计 ---
    e_arch = epic(
        "架构、环境与权限基线",
        "系统基础与安全",
        "阶段二：部署拓扑、环境开通、≥11 类角色权限矩阵、字典基线、综评样板设计。",
        1,
    )
    for title, desc, pri in [
        ("确认部署拓扑与端口清单", "Nginx＋应用＋MySQL8＋OSS；域名证书由校方自购。", 1),
        ("开通测试/生产环境与备份策略", "utf8mb4；异地备份与恢复演练计划（▲）。", 2),
        ("梳理不少于 11 类角色权限矩阵", "菜单/按钮级；数据范围本人/本部门/全部；敏感字段脱敏。", 1),
        ("建立省份/中学/专业等基础字典", "支撑后续配置与筛选项。", 2),
        ("设计综评样板：表单/材料/权重/时间窗", "一场综评＝一条 apply_project；权重合计 100%。", 1),
        ("审计日志与安全基线方案", "登录/审核/导出等关键操作可追溯，保留≥5 年。", 2),
    ]:
        task(title, "系统基础与安全", "S2", e_arch, desc=desc, priority=pri)

    e_ui = epic(
        "品牌视觉与多端一致性",
        "UI品牌视觉",
        "对外端对齐学校官网 VI；招办后台同色板管理端呈现。",
        2,
    )
    for title, desc, pri in [
        ("固化官网主色/强调色/中性色 token", "▲不以第三方配色替代品牌色。", 2),
        ("统一标题/正文字体与版心间距", "PC/H5 响应式；列表白底细线分隔。", 3),
        ("验收对照表：对外端 vs 官网观感", "导航/主按钮/链接/标签色彩对齐。", 2),
    ]:
        task(title, "UI品牌视觉", "S2", e_ui, desc=desc, priority=pri)

    # --- S3 系统实现：按十模块 ---
    e_portal = epic(
        "门户与信息公开",
        "门户与信息公开",
        "栏目/轮播、简章计划、公告公示、报考导引、录取查询入口；CMS 一级分类控制导航。",
    )
    for title, desc, pri in [
        ("配置通知公告/招生活动/信息公开栏目", "一级导航由 CMS enabled 控制；页脚考生/中学入口。", 2),
        ("简章与招生计划发布能力", "与综评/高考查询入口衔接，避免高考报名主 CTA 混淆。", 2),
        ("轮播/Banner 与专题页运营位", "招生季可快速调整。", 3),
    ]:
        task(title, "门户与信息公开", "S3", e_portal, desc=desc, priority=pri)

    e_zhp = epic(
        "综合评价主线",
        "综合评价",
        "项目场次→报名材料→资格审核→缴费准考证→测试成绩→录取衔接；多场次并存。",
        1,
    )
    for title, desc, pri in [
        ("项目与场次配置（多场次/费用/名额/主推）", "状态按时间窗自动切换；下线不影响历史查询。", 1),
        ("▲可视化报名表单与材料清单", "分步填报、草稿续填、诚信承诺、截止前可改；教师代录/Excel/OCR。", 1),
        ("▲身份证/手机号查重", "单条与批量；留痕。", 1),
        ("资格与材料审核工作台", "初审→复审；通过/驳回/补材；意见与材料预览；结果通知。", 1),
        ("▲缴费后准考证与逾期判弃", "微信/支付宝/银联或跳转财务；准考证含考场座位可下载。", 1),
        ("测试安排、分项录分与权重合成公示", "可配置权重（如 631）；入围名单公示。", 2),
        ("▲综评与高考录取分轨 + 双因子查询", "考生号+身份证查询进程与物流单号。", 1),
    ]:
        task(title, "综合评价", "S3", e_zhp, desc=desc, priority=pri, typ="story")

    e_crm = epic(
        "生源 CRM",
        "生源管理",
        "多渠道入库、查重合并、标签分级、认领跟进、转化漏斗、年度归档。",
        1,
    )
    for title, desc, pri in [
        ("▲多渠道生源归集与来源标记", "报名/活动/代录/意向/中学推荐等统一入库。", 1),
        ("▲身份证/手机号查重合并", "自动合并+人工确认，合并留痕。", 1),
        ("标签与意向等级 A+/A/B/C", "档案与联系记录统一。", 2),
        ("认领分工、跟进提醒与转化漏斗", "可关联开放中的综评项目。", 2),
        ("多维分析导出与年度归档只读", "导出留痕。", 3),
    ]:
        task(title, "生源管理", "S3", e_crm, desc=desc, priority=pri, typ="story")

    e_act = epic(
        "招生活动运营",
        "招生活动",
        "开放日/夏冬令营/团体进校同库 type；限额报名、签到核验、线索回流。",
        2,
    )
    for title, desc, pri in [
        ("活动发布：时间地点议程名额", "同库按类型筛选，不拆侧栏。", 2),
        ("▲报满自动停报", "时段独立核算，前后端一致。", 1),
        ("▲团体申请与名单 Excel 校验审批", "审批结果自动通知。", 1),
        ("扫码报名/签到与入校二维码", "可联动门禁；问卷与导出。", 2),
        ("活动线索回流生源库", "标记活动来源，可引导综评报名。", 2),
    ]:
        task(title, "招生活动", "S3", e_act, desc=desc, priority=pri, typ="story")

    e_pay = epic(
        "在线缴费共用能力",
        "在线缴费",
        "订单服务端生成；多渠道支付回调验签；对账退费；状态同步。",
        1,
    )
    for title, desc, pri in [
        ("报名费/测试费下单与签名", "金额服务端计算，防篡改。", 1),
        ("微信/支付宝/银联或财务跳转对接", "回调验签与状态实时同步。", 1),
        ("对账、退费与逾期判弃联动", "与综评状态机衔接。", 2),
    ]:
        task(title, "在线缴费", "S3", e_pay, desc=desc, priority=pri, typ="story")

    e_msg = epic(
        "消息多通道触达",
        "消息通知",
        "短信/站内信/邮件/微信；模板、事件触发、群发、失败重试。",
        2,
    )
    for title, desc, pri in [
        ("通知模板与业务事件绑定", "报名/审核/缴费/准考证/成绩/录取/活动。", 2),
        ("四通道发送与发送日志", "失败自动重试。", 2),
        ("人群群发能力", "招办运营触达。", 3),
    ]:
        task(title, "消息通知", "S3", e_msg, desc=desc, priority=pri)

    e_ai = epic(
        "智能问答（ningharness）",
        "智能问答",
        "嵌入 harness；工具经 Gateway；无 LLM Key 时问答对/检索兜底。",
        2,
    )
    for title, desc, pri in [
        ("▲无大模型 Key 时检索/问答对兜底", "保证演示与招生季咨询不挂。", 1),
        ("知识库维护与未知问题统计", "转人工工单。", 2),
        ("工具只读查 CMS/FAQ/专业/项目", "模型不得直写业务库。", 2),
    ]:
        task(title, "智能问答", "S3", e_ai, desc=desc, priority=pri, typ="story")

    e_gaokao = epic(
        "高考录取分轨",
        "高考录取",
        "名单导入、通知书打印快递、新生移交、对外双因子查询；可与主线并行。",
        2,
    )
    for title, desc, pri in [
        ("录取名单批量导入与逐行校验", "与综评分轨存储统计。", 2),
        ("通知书模板与快递单号", "物流可查。", 2),
        ("新生数据管理与移交", "对接教务/学籍预留。", 3),
        ("对外录取进程查询入口", "考生号+身份证。", 2),
    ]:
        task(title, "高考录取", "S3", e_gaokao, desc=desc, priority=pri, typ="story")

    e_portal_apps = epic(
        "考生/中学三端入口",
        "三端入口",
        "PC Web + 响应式 H5（含微信内置浏览器）+ 独立微信小程序（建设期范围）；同一套 API/账号。",
        1,
    )
    for title, desc, pri in [
        ("▲考生注册登录与隐私授权留痕", "手机号/身份证；短信登录；敏感信息单独授权。", 1),
        ("考生个人中心：报名/缴费/成绩/消息", "断点续填、材料校验、准考证下载。", 1),
        ("中学资质报备与团体进校申请", "▲中学信息审批；名单 Excel；入校二维码。", 2),
        ("多端统一账号体系", "▲不强制独立 App；可对接微信授权。", 1),
        ("考生端微信小程序（UniApp）骨架", "与 Web 共用 Go API；禁止平行业务库。", 2),
    ]:
        task(title, "三端入口", "S3", e_portal_apps, desc=desc, priority=pri, typ="story")

    e_admin = epic(
        "招办后台 2.0 IA 与运营",
        "招办后台与看板",
        "侧栏按 2.0 重组；活动运营单一菜单；工作台指标/待办；领导只读看板。",
        1,
    )
    for title, desc, pri in [
        ("招办后台 IA/权限/壳页落地", "P0：现网页迁入新分组；活动运营不拆侧栏。", 1),
        ("工作台指标卡与待办入口", "生源/综评/活动快捷入口。", 2),
        ("统计分析与校领导只读看板", "转化漏斗、分省、中学 TOP、计划完成率。", 2),
        ("消息模板管理与发送日志页", "与消息模块共用。", 3),
    ]:
        task(title, "招办后台与看板", "S3", e_admin, desc=desc, priority=pri, typ="story")

    e_integ = epic(
        "通道联调与批量配置",
        "集成联调与运维交付",
        "阶段三重点：支付/短信/微信/身份分批联调；剩余项目批量配置。",
        1,
    )
    for title, desc, pri in [
        ("短信/邮件/微信通道联调", "输出联调记录；通道可用性确认。", 1),
        ("支付通道联调与对账样例", "含跳转财务页场景。", 1),
        ("统一身份对接（若资源到位）", "会话超时、失败锁定。", 2),
        ("剩余综评/活动项目批量配置", "样板确认后复制；配置变更留痕。", 2),
    ]:
        task(title, "集成联调与运维交付", "S3", e_integ, desc=desc, priority=pri)

    # --- S4 测试 ---
    e_qa = epic(
        "测试、压测与 UAT",
        "集成联调与运维交付",
        "阶段四：功能/接口/性能（报名与录取查询峰值）/安全/兼容；缺陷台账；UAT。",
        1,
    )
    for title, desc, pri in [
        ("功能与接口测试用例执行", "覆盖十模块主路径与▲条款。", 1),
        ("报名与录取查询峰值压测", "招生季专项保障输入。", 1),
        ("安全测试：越权/脱敏/HTTPS/会话", "招办与专家角色隔离（▲）。", 1),
        ("PC/移动/微信内置浏览器兼容", "近两个大版本主流浏览器。", 2),
        ("组织 UAT 并关闭缺陷台账", "校方业务参与窗口书面约定。", 1),
    ]:
        task(title, "集成联调与运维交付", "S4", e_qa, desc=desc, priority=pri)

    # --- S5 上线 ---
    e_go = epic(
        "上线切换、培训与验收",
        "集成联调与运维交付",
        "阶段五：生产切换、首批培训≥2 次、巡检、验收材料与源码交付。",
        1,
    )
    for title, desc, pri in [
        ("上线准备会与生产切换", "《上线检查单》；尽量避开高峰窗口。", 1),
        ("招办操作培训（≥2 次）", "手册+演示材料；答疑支持。", 2),
        ("上线首周重点巡检与试运行监控", "遗留项收尾。", 1),
        ("验收材料与源代码交付包", "可编译可部署；库脚本/部署/接口/操作/维护手册。", 1),
        ("运维交接：5 年服务机制与招生季保障", "★≥5 年；4–8 月 7×24；团队 客户经理+运维工程师。", 1),
    ]:
        task(title, "集成联调与运维交付", "S5", e_go, desc=desc, priority=pri)

    # backlog / 运维（不进建设期迭代）
    e_ops = epic(
        "验收后 5 年运维服务包",
        "集成联调与运维交付",
        "故障分级响应、巡检备份、内容发布、安全防护、小幅迭代工时、招生季保障。",
        2,
    )
    for title, desc, pri in [
        ("故障响应：紧急 30min/4h，一般 24h", "商务承诺与值班表。", 2),
        ("季度安全检测与年度运行服务报告", "备份恢复演练每年≥1 次（▲）。", 2),
        ("招生季 4–8 月专项保障预案", "压测、扩容协同、应急信息绿色通道。", 1),
    ]:
        task(title, "集成联调与运维交付", None, e_ops, desc=desc, priority=pri)

    # summary
    issues = req("GET", f"/api/issues?projectId={project_id}&rootsOnly=0", token)
    roots = [i for i in issues if not i.get("parentId")]
    children = [i for i in issues if i.get("parentId")]
    print(
        json.dumps(
            {
                "project": "GBU",
                "modules": len(modules),
                "cycles": len(cycles),
                "epics": len(roots),
                "tasks": len(children),
                "totalIssues": len(issues),
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        sys.exit(130)
