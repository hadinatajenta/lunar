<template>
  <div ref="containerRef" class="moon-backdrop" :class="{ 'is-light': !isDark }" aria-hidden="true">
    <canvas ref="canvasRef" class="moon-canvas" />
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
  const surfaceColor = isDark.value ? 0x9ba1ab : 0xc2c7ce
  const shadowColor = isDark.value ? 0x21252c : 0x77808e
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
  left: 58%;
  width: min(78vw, 860px);
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  pointer-events: none;
  z-index: 0;
}

.moon-canvas {
  display: block;
  width: 100%;
  height: 100%;
  opacity: 0.8;
  mask-image: linear-gradient(100deg, transparent 34%, black 68%);
}

.is-light .moon-canvas {
  opacity: 0.2;
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
    width: min(96vw, 620px);
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
