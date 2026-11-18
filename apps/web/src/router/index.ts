import { createRouter, createWebHashHistory } from "vue-router"
import { getToken } from "../lib/api"

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("../views/LoginView.vue"),
      meta: { public: true, title: "登录" },
    },
    {
      path: "/",
      name: "home",
      component: () => import("../views/HomeView.vue"),
      meta: { title: "工作台" },
    },
    {
      path: "/projects",
      name: "projects",
      component: () => import("../views/ProjectsView.vue"),
      meta: { title: "项目" },
    },
    {
      path: "/account",
      name: "account",
      component: () => import("../views/AccountView.vue"),
      meta: { title: "账号" },
    },
    {
      path: "/projects/:id",
      name: "project-detail",
      component: () => import("../views/ProjectDetailView.vue"),
      meta: { title: "项目详情" },
    },
    {
      path: "/mine",
      redirect: { path: "/", query: { scope: "mine" } },
    },
    {
      path: "/inbox",
      redirect: "/",
    },
    {
      path: "/issues",
      redirect: "/",
    },
  ],
})

router.beforeEach((to) => {
  if (to.meta.public) return true
  if (!getToken()) return { name: "login", query: { redirect: to.fullPath } }
  return true
})
