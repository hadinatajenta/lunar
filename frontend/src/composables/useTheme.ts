import { computed, readonly, ref } from "vue"

export type ThemeMode = "dark" | "light"

const STORAGE_KEY = "lunar_theme"

function getInitialTheme(): ThemeMode {
  if (typeof window === "undefined") {
    return "dark"
  }
  const storedTheme = localStorage.getItem(STORAGE_KEY)
  if (storedTheme === "light" || storedTheme === "dark") {
    return storedTheme
  }
  return "dark"
}

function applyThemeToDocument(themeMode: ThemeMode): void {
  if (typeof document !== "undefined") {
    document.documentElement.setAttribute("data-theme", themeMode)
  }
}

const initialTheme = getInitialTheme()
applyThemeToDocument(initialTheme)

const currentTheme = ref<ThemeMode>(initialTheme)

export function useTheme() {
  function setTheme(themeMode: ThemeMode): void {
    currentTheme.value = themeMode
    applyThemeToDocument(themeMode)
    if (typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, themeMode)
    }
  }

  function toggleTheme(): void {
    const nextTheme: ThemeMode = currentTheme.value === "dark" ? "light" : "dark"
    setTheme(nextTheme)
  }

  return {
    theme: readonly(currentTheme),
    isDark: computed(() => currentTheme.value === "dark"),
    toggleTheme,
    setTheme
  }
}
