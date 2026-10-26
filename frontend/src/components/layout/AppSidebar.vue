<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { useAuth } from "../../features/auth/composables/useAuth"
import SidebarNavIcon from "./SidebarNavIcon.vue"
import { useMainSidebar } from "./useMainSidebar"
import { useFocusTrap } from "../../composables/useFocusTrap"

const route = useRoute()
const router = useRouter()
const { user, logout } = useAuth()
const {
  isMenuCollapsed,
  isMobileMenuOpen,
  toggleMenuCollapsed,
  closeMobileMenu,
  resetMobileMenu
} = useMainSidebar()

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

const narrowViewportQuery = window.matchMedia("(max-width: 768px)")

const isNarrowViewport = ref(narrowViewportQuery.matches)
const sidebarElement = ref<HTMLElement | null>(null)
let previousBodyOverflow = ""

const isSidebarInert = computed(() => isNarrowViewport.value && !isMobileMenuOpen.value)

const isModalDrawer = computed(() => isNarrowViewport.value && isMobileMenuOpen.value)

const isNavItemActive = (itemPath: string): boolean =>
  route.path === itemPath || (itemPath !== "/dashboard" && route.path.startsWith(itemPath))

const collapseLabel = computed(() => {
  if (isNarrowViewport.value) {
    return "Close main menu"
  }
  return isMenuCollapsed.value ? "Expand menu" : "Collapse menu"
})

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

const handleDrawerKeydown = (event: KeyboardEvent): void => {
  if (event.key !== "Escape") return
  event.preventDefault()
  closeMobileMenu()
}

const handleCollapseClick = (): void => {
  if (isNarrowViewport.value) {
    closeMobileMenu()
    return
  }
  toggleMenuCollapsed()
}

const handleViewportChange = (event: MediaQueryListEvent): void => {
  isNarrowViewport.value = event.matches
  if (event.matches) return
  resetMobileMenu()
}

const handleSignOut = async () => {
  resetMobileMenu()
  await logout()
  await router.push("/login")
}

const handleNavItemClick = (): void => {
  if (isNarrowViewport.value) {
    resetMobileMenu()
  }
}

watch(isModalDrawer, (isOpen) => {
  if (isOpen) {
    previousBodyOverflow = document.body.style.overflow
    document.body.style.overflow = "hidden"
    return
  }
  document.body.style.overflow = previousBodyOverflow
})

watch(isMobileMenuOpen, (isOpen) => {
  if (isOpen) {
    window.addEventListener("keydown", handleDrawerKeydown, true)
    return
  }
  window.removeEventListener("keydown", handleDrawerKeydown, true)
})

useFocusTrap(sidebarElement, isModalDrawer)

onMounted(() => {
  if (isNarrowViewport.value) resetMobileMenu()
})

narrowViewportQuery.addEventListener("change", handleViewportChange)

onBeforeUnmount(() => {
  narrowViewportQuery.removeEventListener("change", handleViewportChange)
  window.removeEventListener("keydown", handleDrawerKeydown, true)
  document.body.style.overflow = previousBodyOverflow
})
</script>

<template>
  <aside
    id="main-menu"
    ref="sidebarElement"
    class="sidebar"
    :class="{ 'is-collapsed': isMenuCollapsed, 'is-drawer-open': isMobileMenuOpen }"
    :inert="isSidebarInert"
    :role="isModalDrawer ? 'dialog' : undefined"
    :aria-modal="isModalDrawer ? 'true' : undefined"
    aria-label="Main menu"
  >
    <div class="sidebar-header">
      <div class="brand">
        <div class="brand-mark" aria-hidden="true"></div>
        <span class="brand-name">Lunar</span>
      </div>

      <button
        class="collapse-btn"
        type="button"
        :aria-label="collapseLabel"
        :aria-expanded="isNarrowViewport ? undefined : !isMenuCollapsed"
        aria-controls="main-menu"
        @click="handleCollapseClick"
      >
        <svg v-if="isNarrowViewport" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
        <svg v-else-if="isMenuCollapsed" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="m9 6 6 6-6 6"></path>
        </svg>
        <svg v-else aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="m15 6-6 6 6 6"></path>
        </svg>
      </button>
    </div>

    <div class="sidebar-scroll">
      <div class="nav-label">Workspace</div>

      <nav class="nav" aria-label="Main Navigation">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ 'is-active': isNavItemActive(item.path) }"
          :title="isMenuCollapsed ? item.name : undefined"
          @click="handleNavItemClick"
        >
          <span class="nav-icon" aria-hidden="true">
            <SidebarNavIcon :name="item.icon" />
          </span>
          <span class="nav-text">{{ item.name }}</span>
        </RouterLink>
      </nav>
    </div>

    <div class="account">
      <div class="avatar" aria-hidden="true">{{ userInitials }}</div>
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

  <Teleport to="body">
    <div
      v-if="isModalDrawer"
      class="drawer-backdrop"
      data-testid="menu-drawer-backdrop"
      @click="closeMobileMenu"
    ></div>
  </Teleport>
</template>

<style scoped src="./app-sidebar.css"></style>
<style scoped src="./app-sidebar-drawer.css"></style>
