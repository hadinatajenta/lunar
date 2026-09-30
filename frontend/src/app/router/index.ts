import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router"
import { getStoredToken } from "@/lib/http"
import LoginPage from "@/features/auth/pages/LoginPage.vue"
import RegisterPage from "@/features/auth/pages/RegisterPage.vue"
import DashboardPage from "@/features/dashboard/pages/DashboardPage.vue"
import JiraPage from "@/features/jira/pages/JiraPage.vue"
import BitbucketPage from "@/features/bitbucket/pages/BitbucketPage.vue"
import ConfluencePage from "@/features/confluence/pages/ConfluencePage.vue"
import CopilotPage from "@/features/copilot/pages/CopilotPage.vue"
import SettingsPage from "@/features/settings/pages/SettingsPage.vue"

const routes: RouteRecordRaw[] = [
  {
    path: "/login",
    name: "Login",
    component: LoginPage,
    meta: {
      title: "Lunar - Sign in",
      public: true
    }
  },
  {
    path: "/register",
    name: "Register",
    component: RegisterPage,
    meta: {
      title: "Lunar - Create account",
      public: true
    }
  },
  {
    path: "/",
    redirect: "/dashboard"
  },
  {
    path: "/dashboard",
    name: "Dashboard",
    component: DashboardPage,
    meta: {
      title: "Lunar - Operations Overview"
    }
  },
  {
    path: "/jira",
    name: "Jira",
    component: JiraPage,
    meta: {
      title: "Lunar - Jira BRI"
    }
  },
  {
    path: "/bitbucket",
    name: "Bitbucket",
    component: BitbucketPage,
    meta: {
      title: "Lunar - Bitbucket BRI"
    }
  },
  {
    path: "/confluence",
    name: "Confluence",
    component: ConfluencePage,
    meta: {
      title: "Lunar - Confluence BRI"
    }
  },
  {
    path: "/copilot",
    name: "Copilot",
    component: CopilotPage,
    meta: {
      title: "Lunar - AI Copilot"
    }
  },
  {
    path: "/settings",
    name: "Settings",
    component: SettingsPage,
    meta: {
      title: "Lunar - Settings"
    }
  },
  {
    path: "/forgot-password",
    redirect: "/login"
  },
  {
    path: "/request-access",
    redirect: "/register"
  },
  {
    path: "/:pathMatch(.*)*",
    redirect: "/dashboard"
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  if (to.meta.title && typeof to.meta.title === "string") {
    document.title = to.meta.title
  }

  const token = getStoredToken()
  const isPublic = Boolean(to.meta.public)

  if (!isPublic && !token) {
    return { path: "/login" }
  }

  if ((to.path === "/login" || to.path === "/register") && token) {
    return { path: "/dashboard" }
  }
})

export default router
