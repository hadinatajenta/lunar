<script setup lang="ts">
import { computed } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useAuth } from "../../features/auth/composables/useAuth"

const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()

const navItems = [
  {
    name: "Overview",
    path: "/dashboard",
    icon: "dashboard"
  },
  {
    name: "Jira BRI",
    path: "/jira",
    icon: "jira"
  },
  {
    name: "Bitbucket BRI",
    path: "/bitbucket",
    icon: "bitbucket"
  },
  {
    name: "Confluence BRI",
    path: "/confluence",
    icon: "confluence"
  },
  {
    name: "Microservices BRI",
    path: "/microservices",
    icon: "microservices"
  },
  {
    name: "Copilot",
    path: "/copilot",
    icon: "copilot"
  },
  {
    name: "Settings",
    path: "/settings",
    icon: "settings"
  }
]

const userInitials = computed(() => {
  if (!user.value || !user.value.full_name) {
    return "LD"
  }
  const parts = user.value.full_name.trim().split(" ")
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return user.value.full_name.slice(0, 2).toUpperCase()
})

const handleSignOut = async () => {
  await logout()
  await router.push("/login")
}
</script>

<template>
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark" aria-hidden="true"></div>
      <span class="brand-name">Lunar</span>
    </div>

    <div class="nav-label">Workspace</div>

    <nav class="nav" aria-label="Main Navigation">
      <RouterLink
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="nav-item"
        :class="{ 'is-active': route.path === item.path || (item.path !== '/dashboard' && route.path.startsWith(item.path)) }"
      >
        <span class="nav-icon" aria-hidden="true">
          <svg v-if="item.icon === 'dashboard'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <rect x="4" y="4" width="6" height="6" rx="1"></rect>
            <rect x="14" y="4" width="6" height="6" rx="1"></rect>
            <rect x="4" y="14" width="6" height="6" rx="1"></rect>
            <rect x="14" y="14" width="6" height="6" rx="1"></rect>
          </svg>
          <svg v-else-if="item.icon === 'jira'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M7 4h10l-3 4H7z"></path>
            <path d="M7 20h10l-3-4H7z"></path>
            <path d="M17 4v6a4 4 0 0 1-4 4H7"></path>
          </svg>
          <svg v-else-if="item.icon === 'bitbucket'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M5 5h14l-2 14H7z"></path>
            <path d="M9 9h6"></path>
            <path d="M10 9l1 7"></path>
          </svg>
          <svg v-else-if="item.icon === 'confluence'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 16c3-5 5-8 12-8"></path>
            <path d="M8 18c2-4 4-6 10-6"></path>
          </svg>
          <svg v-else-if="item.icon === 'microservices'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 3l9 4.5v9L12 21l-9-4.5v-9z"></path>
            <path d="M3 7.5 12 12l9-4.5"></path>
            <path d="M12 12v9"></path>
          </svg>
          <svg v-else-if="item.icon === 'copilot'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 8a6 6 0 0 1 12 0v6a4 4 0 0 1-4 4H10a4 4 0 0 1-4-4z"></path>
            <path d="M9 11h.01"></path>
            <path d="M15 11h.01"></path>
            <path d="M9 15c2 1 4 1 6 0"></path>
          </svg>
          <svg v-else-if="item.icon === 'settings'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 3v2"></path>
            <path d="M12 19v2"></path>
            <path d="m4.93 4.93 1.41 1.41"></path>
            <path d="m17.66 17.66 1.41 1.41"></path>
            <path d="M3 12h2"></path>
            <path d="M19 12h2"></path>
            <path d="m4.93 19.07 1.41-1.41"></path>
            <path d="m17.66 6.34 1.41-1.41"></path>
            <circle cx="12" cy="12" r="4"></circle>
          </svg>
        </span>
        <span>{{ item.name }}</span>
      </RouterLink>
    </nav>

    <div class="sidebar-spacer"></div>

    <div class="account">
      <div class="avatar">{{ userInitials }}</div>
      <div class="account-copy">
        <div class="account-name">{{ user?.full_name || "Lunar Developer" }}</div>
        <div class="account-email">{{ user?.email || "developer@lunar.dev" }}</div>
      </div>
      <button class="logout-btn" type="button" title="Sign out" aria-label="Sign out" @click="handleSignOut">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
          <polyline points="16 17 21 12 16 7"></polyline>
          <line x1="21" y1="12" x2="9" y2="12"></line>
        </svg>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  position: sticky;
  top: 0;
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 18px 12px;
  border-right: 1px solid var(--border);
  background: var(--sidebar);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px 28px;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.brand-mark {
  position: relative;
  width: 25px;
  height: 25px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border-strong);
  border-radius: 50%;
}

.brand-mark::before {
  content: "";
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text);
  box-shadow: 0 0 16px rgba(125, 125, 125, 0.22);
}

.brand-name {
  color: var(--text);
}

.nav-label {
  padding: 0 10px 8px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.11em;
  text-transform: uppercase;
}

.nav {
  display: grid;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 38px;
  padding: 0 10px;
  border-radius: 9px;
  color: var(--muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 180ms ease,
    color 180ms ease;
}

.nav-item:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.nav-item.is-active {
  background: var(--surface-raised);
  color: var(--text);
  font-weight: 600;
}

.nav-icon {
  width: 16px;
  height: 16px;
  display: grid;
  place-items: center;
}

.nav-icon svg {
  width: 15px;
  height: 15px;
}

.sidebar-spacer {
  flex: 1;
}

.account {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 14px 9px 4px;
  border-top: 1px solid var(--border);
}

.avatar {
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border-strong);
  border-radius: 9px;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.account-copy {
  min-width: 0;
  flex: 1;
}

.account-name {
  overflow: hidden;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-email {
  overflow: hidden;
  margin-top: 2px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.logout-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease;
  flex-shrink: 0;
}

.logout-btn svg {
  width: 14px;
  height: 14px;
}

.logout-btn:hover {
  background: var(--surface-hover);
  color: var(--danger);
}
</style>
