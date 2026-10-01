export const VERTEX_SHADER = `
varying vec3 vWorldNormal;
varying vec3 vWorldPosition;
varying float vHeight;

float hash21(vec2 position) {
  vec3 fractional = fract(vec3(position.xyx) * 0.1031);
  fractional += dot(fractional, fractional.yzx + 33.33);
  return fract((fractional.x + fractional.y) * fractional.z);
}

float hash31(vec3 position) {
  position = fract(position * 0.3183099 + vec3(0.71, 0.113, 0.419));
  position *= 17.0;
  return fract(position.x * position.y * position.z * (position.x + position.y + position.z));
}

float valueNoise(vec3 position) {
  vec3 cell = floor(position);
  vec3 offset = fract(position);
  vec3 blend = offset * offset * (3.0 - 2.0 * offset);

  float corner000 = hash31(cell + vec3(0.0, 0.0, 0.0));
  float corner100 = hash31(cell + vec3(1.0, 0.0, 0.0));
  float corner010 = hash31(cell + vec3(0.0, 1.0, 0.0));
  float corner110 = hash31(cell + vec3(1.0, 1.0, 0.0));
  float corner001 = hash31(cell + vec3(0.0, 0.0, 1.0));
  float corner101 = hash31(cell + vec3(1.0, 0.0, 1.0));
  float corner011 = hash31(cell + vec3(0.0, 1.0, 1.0));
  float corner111 = hash31(cell + vec3(1.0, 1.0, 1.0));

  return mix(
    mix(mix(corner000, corner100, blend.x), mix(corner010, corner110, blend.x), blend.y),
    mix(mix(corner001, corner101, blend.x), mix(corner011, corner111, blend.x), blend.y),
    blend.z
  );
}

float fractalNoise(vec3 position) {
  float total = 0.0;
  float amplitude = 0.5;
  vec3 samplePoint = position;

  for (int octave = 0; octave < 3; octave++) {
    total += valueNoise(samplePoint) * amplitude;
    samplePoint *= 2.07;
    amplitude *= 0.5;
  }

  return total;
}

vec2 cubeFaceUv(vec3 direction) {
  vec3 magnitudes = abs(direction);
  vec3 safeMagnitudes = max(magnitudes, vec3(0.0001));
  vec2 uvX = direction.yz / safeMagnitudes.x;
  vec2 uvY = direction.xz / safeMagnitudes.y;
  vec2 uvZ = direction.xy / safeMagnitudes.z;
  bool useXFace = magnitudes.x >= magnitudes.y && magnitudes.x >= magnitudes.z;
  bool useYFace = magnitudes.y >= magnitudes.z;
  return useXFace ? uvX : (useYFace ? uvY : uvZ);
}

float craterLayer(vec3 direction, float cellsPerFace, float baseRadius) {
  vec2 uv = cubeFaceUv(direction) * cellsPerFace;
  vec2 baseCell = floor(uv);
  float deepestDepression = 0.0;
  float highestRim = 0.0;

  for (int xOffset = -1; xOffset <= 1; xOffset++) {
    for (int yOffset = -1; yOffset <= 1; yOffset++) {
      vec2 neighbor = baseCell + vec2(float(xOffset), float(yOffset));
      vec2 jitter = vec2(hash21(neighbor + 3.17), hash21(neighbor + 17.71)) - 0.5;
      vec2 craterCenter = neighbor + vec2(0.5) + jitter * 0.72;
      float radius = baseRadius * (0.6 + 0.4 * hash21(neighbor + 41.37));
      float normalizedDistance = length(uv - craterCenter) / radius;

      if (normalizedDistance > 1.35) {
        continue;
      }

      float depression = 1.0 - smoothstep(0.0, 0.82, normalizedDistance);
      depression *= depression;
      float rim = smoothstep(0.68, 1.02, normalizedDistance)
        * (1.0 - smoothstep(1.02, 1.34, normalizedDistance));

      deepestDepression = max(deepestDepression, depression);
      highestRim = max(highestRim, rim);
    }
  }

  return highestRim * 0.68 - deepestDepression * 0.95;
}

float surfaceHeight(vec3 direction) {
  float height = (fractalNoise(direction * 2.1) - 0.5) * 0.004;
  height += (fractalNoise(direction * 5.6) - 0.5) * 0.0025;
  height += craterLayer(direction, 6.0, 0.34) * 0.014;
  height += craterLayer(direction, 13.0, 0.18) * 0.007;
  height += craterLayer(direction, 27.0, 0.1) * 0.0035;
  return height;
}

void main() {
  vec3 direction = normalize(position);
  float centerHeight = surfaceHeight(direction);
  vec3 displacedCenter = direction * (1.0 + centerHeight);

  vec3 referenceAxis = abs(direction.y) < 0.99 ? vec3(0.0, 1.0, 0.0) : vec3(1.0, 0.0, 0.0);
  vec3 tangentDirection = normalize(cross(referenceAxis, direction));
  vec3 bitangentDirection = normalize(cross(direction, tangentDirection));

  float sampleDelta = 0.007;
  vec3 tangentSample = normalize(direction + tangentDirection * sampleDelta);
  vec3 bitangentSample = normalize(direction + bitangentDirection * sampleDelta);

  vec3 displacedTangent = tangentSample * (1.0 + surfaceHeight(tangentSample));
  vec3 displacedBitangent = bitangentSample * (1.0 + surfaceHeight(bitangentSample));

  vec3 objectNormal = normalize(
    cross(displacedTangent - displacedCenter, displacedBitangent - displacedCenter)
  );
  if (dot(objectNormal, direction) < 0.0) {
    objectNormal = -objectNormal;
  }

  vec4 worldPosition = modelMatrix * vec4(displacedCenter, 1.0);

  vHeight = centerHeight;
  vWorldNormal = normalize(mat3(modelMatrix) * objectNormal);
  vWorldPosition = worldPosition.xyz;

  gl_Position = projectionMatrix * viewMatrix * worldPosition;
}
`

export const FRAGMENT_SHADER = `
uniform vec3 uLightDirection;
uniform vec3 uSurfaceColor;
uniform vec3 uShadowColor;
uniform float uAmbientStrength;

varying vec3 vWorldNormal;
varying vec3 vWorldPosition;
varying float vHeight;

void main() {
  vec3 surfaceNormal = normalize(vWorldNormal);
  vec3 lightDirection = normalize(uLightDirection);
  vec3 viewDirection = normalize(cameraPosition - vWorldPosition);

  float lambert = dot(surfaceNormal, lightDirection);
  float directLight = clamp(lambert, 0.0, 1.0);
  float terminator = smoothstep(0.0, 0.38, lambert);

  vec3 surface = mix(uShadowColor, uSurfaceColor, terminator);
  surface *= uAmbientStrength + (1.0 - uAmbientStrength) * directLight;
  surface *= 1.0 + clamp(vHeight, -0.12, 0.12) * 1.4;

  float fresnel = pow(1.0 - max(dot(surfaceNormal, viewDirection), 0.0), 3.4);
  surface += vec3(0.56, 0.64, 0.86) * fresnel * 0.17;

  gl_FragColor = vec4(surface, 1.0);
}
`
