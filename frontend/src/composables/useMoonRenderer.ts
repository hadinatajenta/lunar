import type {
  Mesh,
  PerspectiveCamera,
  ShaderMaterial,
  WebGLRenderer
} from "three"
import { FRAGMENT_SHADER, VERTEX_SHADER } from "@/lib/moonShaders"

type ThreeModule = typeof import("three")

export interface MoonRenderer {
  setSurfaceColors: (surfaceColor: number, shadowColor: number) => void
  dispose: () => void
}

const SPHERE_RADIUS = 1.0
const SPHERE_DETAIL = 8
const CAMERA_DISTANCE = 5.4
const CAMERA_FOV = 30
const ROTATION_SPEED_RADIANS = 0.04
const MAX_PIXEL_RATIO = 2
const RESIZE_DEBOUNCE_MS = 140
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)"

function createMoonMaterial(three: ThreeModule): ShaderMaterial {
  return new three.ShaderMaterial({
    vertexShader: VERTEX_SHADER,
    fragmentShader: FRAGMENT_SHADER,
    uniforms: {
      uLightDirection: { value: new three.Vector3(-0.62, 0.38, 0.52).normalize() },
      uSurfaceColor: { value: new three.Color(0x7f7a72).convertLinearToSRGB() },
      uShadowColor: { value: new three.Color(0x1a1612).convertLinearToSRGB() },
      uAmbientStrength: { value: 0.055 },
      uMariaColor: { value: new three.Color(0x44413c).convertLinearToSRGB() },
      uHighlandColor: { value: new three.Color(0x8a857c).convertLinearToSRGB() },
      uCraterColor: { value: new three.Color(0x36332e).convertLinearToSRGB() },
      uDepthStrength: { value: 0.34 }
    }
  })
}

function createMoonMesh(three: ThreeModule): Mesh {
  const geometry = new three.IcosahedronGeometry(SPHERE_RADIUS, SPHERE_DETAIL)
  return new three.Mesh(geometry, createMoonMaterial(three))
}

function createRenderer(three: ThreeModule, canvas: HTMLCanvasElement): WebGLRenderer {
  const renderer = new three.WebGLRenderer({
    canvas,
    antialias: true,
    alpha: true,
    powerPreference: "low-power"
  })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, MAX_PIXEL_RATIO))
  return renderer
}

function applyCanvasSize(
  renderer: WebGLRenderer,
  camera: PerspectiveCamera,
  width: number,
  height: number
): void {
  if (width === 0 || height === 0) {
    return
  }
  renderer.setSize(width, height, false)
  camera.aspect = width / height
  camera.updateProjectionMatrix()
}

function prefersReducedMotion(): boolean {
  return window.matchMedia(REDUCED_MOTION_QUERY).matches
}

export async function createMoonRenderer(
  canvas: HTMLCanvasElement
): Promise<MoonRenderer> {
  const three = await import("three")
  const renderer = createRenderer(three, canvas)
  const scene = new three.Scene()
  const camera = new three.PerspectiveCamera(CAMERA_FOV, 1, 0.1, 100)
  const mesh = createMoonMesh(three)

  camera.position.z = CAMERA_DISTANCE
  scene.add(mesh)

  const container = canvas.parentElement
  let animationFrameId = 0
  let resizeTimerId = 0
  let isIntersecting = true
  let isDocumentVisible = !document.hidden
  let isDisposed = false

  function renderFrame(): void {
    applyCanvasSize(
      renderer,
      camera,
      canvas.clientWidth,
      canvas.clientHeight
    )
    renderer.render(scene, camera)
  }

  function advanceRotation(deltaSeconds: number): void {
    mesh.rotation.y += ROTATION_SPEED_RADIANS * deltaSeconds
    mesh.rotation.x = Math.sin(mesh.rotation.y * 0.35) * 0.12
  }

  function tick(previousTimestamp: number, timestamp: number): void {
    if (isDisposed) {
      return
    }
    const deltaSeconds = Math.min((timestamp - previousTimestamp) / 1000, 0.1)
    advanceRotation(deltaSeconds)
    renderFrame()
    animationFrameId = window.requestAnimationFrame((nextTimestamp) => tick(timestamp, nextTimestamp))
  }

  function stopLoop(): void {
    if (animationFrameId !== 0) {
      window.cancelAnimationFrame(animationFrameId)
      animationFrameId = 0
    }
  }

  function renderStaticFrame(): void {
    mesh.rotation.y = 0.6
    renderFrame()
  }

  function syncLoopState(): void {
    if (isDisposed) {
      return
    }
    const shouldRender = isIntersecting && isDocumentVisible
    if (!shouldRender) {
      stopLoop()
      return
    }
    if (animationFrameId === 0) {
      if (prefersReducedMotion()) {
        renderStaticFrame()
        return
      }
      animationFrameId = window.requestAnimationFrame((timestamp) => tick(timestamp, timestamp))
    }
  }

  function handleVisibilityChange(): void {
    isDocumentVisible = !document.hidden
    syncLoopState()
  }

  const resizeObserver = new ResizeObserver(() => {
    window.clearTimeout(resizeTimerId)
    resizeTimerId = window.setTimeout(renderFrame, RESIZE_DEBOUNCE_MS)
  })

  const intersectionObserver = new IntersectionObserver((entries) => {
    const entry = entries[0]
    if (!entry) {
      return
    }
    isIntersecting = entry.isIntersecting
    syncLoopState()
  })

  if (container) {
    resizeObserver.observe(container)
  }
  intersectionObserver.observe(canvas)
  document.addEventListener("visibilitychange", handleVisibilityChange)

  renderStaticFrame()
  syncLoopState()

  return {
    setSurfaceColors(surfaceColor: number, shadowColor: number): void {
      const material = mesh.material as ShaderMaterial
      material.uniforms.uSurfaceColor.value.setHex(surfaceColor).convertLinearToSRGB()
      material.uniforms.uShadowColor.value.setHex(shadowColor).convertLinearToSRGB()
      material.needsUpdate = true
      renderFrame()
    },
    dispose(): void {
      isDisposed = true
      stopLoop()
      window.clearTimeout(resizeTimerId)
      resizeObserver.disconnect()
      intersectionObserver.disconnect()
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      scene.remove(mesh)
      mesh.geometry.dispose()
      const material = mesh.material
      if (Array.isArray(material)) {
        material.forEach((entry) => entry.dispose())
        return
      }
      material.dispose()
      renderer.dispose()
    }
  }
}
