<template>
  <div ref="containerRef" class="moon-backdrop" :class="{ 'is-light': !isDark }" aria-hidden="true">
    <canvas ref="canvasRef" class="moon-canvas" />
    <div class="moon-veil" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue"
import { createMoonRenderer, type MoonRenderer } from "@/composables/useMoonRenderer"
import { useTheme } from "@/composables/useTheme"

const containerRef = ref<HTMLDivElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)

const { isDark } = useTheme()

let renderer: MoonRenderer | null = null
let isUnmounted = false

function applySurfaceColors(): void {
  const surfaceColor = isDark.value ? 0x7f7a72 : 0x9a948a
  const shadowColor = isDark.value ? 0x1a1612 : 0x4a463f
  renderer?.setSurfaceColors(surfaceColor, shadowColor)
}

onMounted(async () => {
  const canvas = canvasRef.value
  if (!canvas) {
    return
  }
  const createdRenderer = await createMoonRenderer(canvas)
  if (isUnmounted) {
    createdRenderer.dispose()
    return
  }
  renderer = createdRenderer
  applySurfaceColors()
})

watch(isDark, applySurfaceColors)

onBeforeUnmount(() => {
  isUnmounted = true
  renderer?.dispose()
  renderer = null
})
</script>

<style scoped>
.moon-backdrop {
  position: absolute;
  top: 50%;
  left: 54%;
  width: min(78vw, 860px, 100vh, 100dvh);
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  pointer-events: none;
  z-index: 0;
}

.moon-canvas {
  display: block;
  width: 100%;
  height: 100%;
  opacity: 0.72;
  mask-image: linear-gradient(
    100deg,
    transparent 4%,
    rgba(0, 0, 0, 0.45) 18%,
    black 42%
  );
}

.moon-veil {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(
    100deg,
    var(--bg) 0%,
    var(--bg) 14%,
    color-mix(in srgb, var(--bg) 86%, transparent) 34%,
    color-mix(in srgb, var(--bg) 34%, transparent) 60%,
    transparent 80%
  );
}

.is-light .moon-veil {
  display: none;
}

.is-light .moon-canvas {
  opacity: 0.26;
  mask-image: radial-gradient(
    circle at 66% 50%,
    black 0%,
    black 40%,
    transparent 76%
  );
}

@media (max-width: 980px) {
  .moon-backdrop {
    top: 34%;
    left: 62%;
    width: min(96vw, 620px, 100vh, 100dvh);
  }

  .moon-canvas {
    opacity: 0.5;
  }

  .is-light .moon-canvas {
    opacity: 0.16;
  }
}

@media (prefers-reduced-motion: reduce) {
  .moon-canvas {
    opacity: 0.3;
  }
}
</style>
