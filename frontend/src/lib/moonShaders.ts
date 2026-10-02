export const VERTEX_SHADER = `
varying vec3 vWorldNormal;
varying vec3 vWorldPosition;
varying vec3 vBumpTangent;
varying vec3 vBumpBitangent;

const float CRATER_LAYER_LARGE_SCALE = 6.0;
const float CRATER_LAYER_LARGE_RADIUS = 0.5333;
const float CRATER_LAYER_MEDIUM_SCALE = 18.0;
const float CRATER_LAYER_MEDIUM_RADIUS = 0.1778;
const float CRATER_LAYER_FINE_SCALE = 44.0;
const float CRATER_LAYER_FINE_RADIUS = 0.0727;

const float CRATER_LARGE_DENSITY = 0.42;
const float CRATER_MEDIUM_DENSITY = 0.54;
const float CRATER_FINE_DENSITY = 0.64;

const float CRATER_CENTRE_JITTER = 0.25;

const float NORMAL_SAMPLE_DELTA = 0.030;

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

float fractalNoise(vec3 position, int octaves, float lacunarity, float gain) {
  float total = 0.0;
  float amplitude = 0.5;
  vec3 samplePoint = position;

  for (int octave = 0; octave < 6; octave++) {
    if (octave >= octaves) break;
    total += valueNoise(samplePoint) * amplitude;
    samplePoint *= lacunarity;
    amplitude *= gain;
  }

  return total;
}

float craterHeight(vec3 direction, float scale, float baseRadius, float density, float sizeBias) {
  vec3 scaledDirection = direction * scale;
  vec3 baseCell = floor(scaledDirection);
  float height = 0.0;

  for (int x = -1; x <= 1; x++) {
    for (int y = -1; y <= 1; y++) {
      for (int z = -1; z <= 1; z++) {
        vec3 cell = baseCell + vec3(float(x), float(y), float(z));
        if (hash31(cell * 1.37 + vec3(11.3, 7.9, 3.1)) > density) continue;

        vec3 jitter = vec3(
          hash31(cell + vec3(3.17, 0.0, 0.0)),
          hash31(cell + vec3(0.0, 17.71, 0.0)),
          hash31(cell + vec3(0.0, 0.0, 41.37))
        ) - 0.5;

        vec3 craterCenter = cell + vec3(0.5) + jitter * CRATER_CENTRE_JITTER * 2.0;
        float sizeRoll = hash31(cell + vec3(71.9, 23.7, 5.3));
        float craterRadius = baseRadius * mix(0.75, 1.85, pow(sizeRoll, sizeBias));
        float craterDepth = 0.50 + 0.85 * hash31(cell + vec3(9.1, 53.3, 29.7));

        float normalizedDistance = length(scaledDirection - craterCenter) / craterRadius;
        if (normalizedDistance > 1.9) continue;

        float isFloor = 1.0 - smoothstep(0.62, 0.90, normalizedDistance);
        float bowl = isFloor * isFloor;
        float isRimBand = smoothstep(0.72, 0.96, normalizedDistance)
          * (1.0 - smoothstep(1.10, 1.62, normalizedDistance));
        float rim = isRimBand * (1.0 - isRimBand) * 4.0;

        height += rim - bowl * craterDepth;
      }
    }
  }

  return height;
}

float sharpenField(float value) {
  return clamp(value * 1.3, -1.0, 1.0);
}

float craterField(vec3 direction) {
  float field = 0.0;
  field += craterHeight(direction, CRATER_LAYER_LARGE_SCALE, CRATER_LAYER_LARGE_RADIUS, CRATER_LARGE_DENSITY, 1.70) * 0.34;
  field += craterHeight(direction, CRATER_LAYER_MEDIUM_SCALE, CRATER_LAYER_MEDIUM_RADIUS, CRATER_MEDIUM_DENSITY, 1.35) * 0.40;
  field += craterHeight(direction, CRATER_LAYER_FINE_SCALE, CRATER_LAYER_FINE_RADIUS, CRATER_FINE_DENSITY, 1.15) * 0.26;
  return sharpenField(field);
}

float mariaEdgeWarp(vec3 direction) {
  return (fractalNoise(direction * 3.1, 3, 2.05, 0.5) - 0.5) * 0.16;
}

float mariaField(vec3 direction) {
  float maria = 0.0;
  float warp = mariaEdgeWarp(direction);

  maria = max(maria, smoothstep(0.66 + warp, 0.20 + warp, length(direction - normalize(vec3(0.16, 0.22, 0.94)))));
  maria = max(maria, smoothstep(0.52 + warp, 0.18 + warp, length(direction - normalize(vec3(-0.26, 0.36, 0.88)))));
  maria = max(maria, smoothstep(0.44 + warp, 0.16 + warp, length(direction - normalize(vec3(0.40, -0.04, 0.90)))));
  maria = max(maria, smoothstep(0.38 + warp, 0.14 + warp, length(direction - normalize(vec3(-0.10, -0.26, 0.94)))));
  maria = max(maria, smoothstep(0.60 + warp, 0.26 + warp, length(direction - normalize(vec3(-0.60, -0.24, 0.72)))));
  maria = max(maria, smoothstep(0.44 + warp, 0.18 + warp, length(direction - normalize(vec3(0.52, 0.52, 0.60)))));
  maria = max(maria, smoothstep(0.34 + warp, 0.12 + warp, length(direction - normalize(vec3(0.60, 0.16, 0.74)))));

  return clamp(maria, 0.0, 1.0);
}

float surfaceHeight(vec3 direction) {
  float height = (fractalNoise(direction * 2.4, 4, 2.1, 0.48) - 0.5) * 0.0035;
  height += (fractalNoise(direction * 7.5, 3, 2.0, 0.50) - 0.5) * 0.0014;
  height += craterField(direction) * 0.0110;
  height -= mariaField(direction) * 0.0025;
  return height;
}

void main() {
  vec3 direction = normalize(position);
  vec3 referenceAxis = abs(direction.y) < 0.99 ? vec3(0.0, 1.0, 0.0) : vec3(1.0, 0.0, 0.0);

  vec3 tangentDirection = normalize(cross(referenceAxis, direction));
  vec3 bitangentDirection = cross(direction, tangentDirection);

  float centerHeight = surfaceHeight(direction);
  vec3 displacedCenter = direction * (1.0 + centerHeight);

  vec3 tangentSample = normalize(direction + tangentDirection * NORMAL_SAMPLE_DELTA);
  vec3 bitangentSample = normalize(direction + bitangentDirection * NORMAL_SAMPLE_DELTA);

  vec3 displacedTangent = tangentSample * (1.0 + surfaceHeight(tangentSample));
  vec3 displacedBitangent = bitangentSample * (1.0 + surfaceHeight(bitangentSample));

  vec3 objectNormal = normalize(
    cross(displacedTangent - displacedCenter, displacedBitangent - displacedCenter)
  );
  if (dot(objectNormal, direction) < 0.0) {
    objectNormal = -objectNormal;
  }

  vec4 worldPosition = modelMatrix * vec4(displacedCenter, 1.0);

  vWorldNormal = normalize(mat3(modelMatrix) * objectNormal);
  vWorldPosition = worldPosition.xyz;
  vBumpTangent = tangentDirection;
  vBumpBitangent = bitangentDirection;

  gl_Position = projectionMatrix * viewMatrix * worldPosition;
}
`

export const FRAGMENT_SHADER = `
uniform vec3 uLightDirection;
uniform vec3 uSurfaceColor;
uniform vec3 uShadowColor;
uniform float uAmbientStrength;
uniform vec3 uMariaColor;
uniform vec3 uHighlandColor;
uniform vec3 uCraterColor;
uniform float uDepthStrength;

varying vec3 vWorldNormal;
varying vec3 vWorldPosition;
varying vec3 vBumpTangent;
varying vec3 vBumpBitangent;

const float TERMINATOR_SOFTNESS = 0.16;
const float SPECULAR_STRENGTH = 0.018;
const float LIMB_GLOW_STRENGTH = 0.018;
const float CRATER_DARKENING = 0.20;
const float CRATER_BRIGHTENING = 0.14;
const float MARIA_DARKENING = 0.60;
const float CRATER_FLOOR_SHADING = 0.72;
const float BUMP_STRENGTH = 0.110;
const float MICRO_SAMPLE_DELTA = 0.0022;
const float BUMP_GAIN = 1.60;
const float LAYER_BUMP_AMPLITUDE = 0.38;
const float MICRO_BUMP_AMPLITUDE = 0.45;
const float CRATER_LAYER_LARGE_SCALE = 6.0;
const float CRATER_LAYER_LARGE_RADIUS = 0.5333;
const float CRATER_LAYER_MEDIUM_SCALE = 18.0;
const float CRATER_LAYER_MEDIUM_RADIUS = 0.1778;
const float CRATER_LAYER_FINE_SCALE = 44.0;
const float CRATER_LAYER_FINE_RADIUS = 0.0727;
const float CRATER_LARGE_DENSITY = 0.42;
const float CRATER_MEDIUM_DENSITY = 0.54;
const float CRATER_FINE_DENSITY = 0.64;
const float CRATER_CENTRE_JITTER = 0.25;

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

float fractalNoise(vec3 position, int octaves, float lacunarity, float gain) {
  float total = 0.0;
  float amplitude = 0.5;
  vec3 samplePoint = position;

  for (int octave = 0; octave < 6; octave++) {
    if (octave >= octaves) break;
    total += valueNoise(samplePoint) * amplitude;
    samplePoint *= lacunarity;
    amplitude *= gain;
  }

  return total;
}

float craterHeight(vec3 direction, float scale, float baseRadius, float density, float sizeBias) {
  vec3 scaledDirection = direction * scale;
  vec3 baseCell = floor(scaledDirection);
  float height = 0.0;

  for (int x = -1; x <= 1; x++) {
    for (int y = -1; y <= 1; y++) {
      for (int z = -1; z <= 1; z++) {
        vec3 cell = baseCell + vec3(float(x), float(y), float(z));
        if (hash31(cell * 1.37 + vec3(11.3, 7.9, 3.1)) > density) continue;

        vec3 jitter = vec3(
          hash31(cell + vec3(3.17, 0.0, 0.0)),
          hash31(cell + vec3(0.0, 17.71, 0.0)),
          hash31(cell + vec3(0.0, 0.0, 41.37))
        ) - 0.5;

        vec3 craterCenter = cell + vec3(0.5) + jitter * CRATER_CENTRE_JITTER * 2.0;
        float sizeRoll = hash31(cell + vec3(71.9, 23.7, 5.3));
        float craterRadius = baseRadius * mix(0.75, 1.85, pow(sizeRoll, sizeBias));
        float craterDepth = 0.50 + 0.85 * hash31(cell + vec3(9.1, 53.3, 29.7));

        float normalizedDistance = length(scaledDirection - craterCenter) / craterRadius;
        if (normalizedDistance > 1.9) continue;

        float isFloor = 1.0 - smoothstep(0.62, 0.90, normalizedDistance);
        float bowl = isFloor * isFloor;
        float isRimBand = smoothstep(0.72, 0.96, normalizedDistance)
          * (1.0 - smoothstep(1.10, 1.62, normalizedDistance));
        float rim = isRimBand * (1.0 - isRimBand) * 4.0;

        height += rim - bowl * craterDepth;
      }
    }
  }

  return height;
}

float sharpenField(float value) {
  return clamp(value * 1.3, -1.0, 1.0);
}


float mariaEdgeWarp(vec3 direction) {
  return (fractalNoise(direction * 3.1, 3, 2.05, 0.5) - 0.5) * 0.16;
}

float mariaField(vec3 direction) {
  float maria = 0.0;
  float warp = mariaEdgeWarp(direction);

  maria = max(maria, smoothstep(0.66 + warp, 0.20 + warp, length(direction - normalize(vec3(0.16, 0.22, 0.94)))));
  maria = max(maria, smoothstep(0.52 + warp, 0.18 + warp, length(direction - normalize(vec3(-0.26, 0.36, 0.88)))));
  maria = max(maria, smoothstep(0.44 + warp, 0.16 + warp, length(direction - normalize(vec3(0.40, -0.04, 0.90)))));
  maria = max(maria, smoothstep(0.38 + warp, 0.14 + warp, length(direction - normalize(vec3(-0.10, -0.26, 0.94)))));
  maria = max(maria, smoothstep(0.60 + warp, 0.26 + warp, length(direction - normalize(vec3(-0.60, -0.24, 0.72)))));
  maria = max(maria, smoothstep(0.44 + warp, 0.18 + warp, length(direction - normalize(vec3(0.52, 0.52, 0.60)))));
  maria = max(maria, smoothstep(0.34 + warp, 0.12 + warp, length(direction - normalize(vec3(0.60, 0.16, 0.74)))));

  return clamp(maria, 0.0, 1.0);
}

float surfaceRoughness(vec3 direction) {
  float relief = (valueNoise(direction * 260.0) - 0.5) * 0.40;
  relief += (valueNoise(direction * 660.0) - 0.5) * 0.22;
  return relief;
}

float layerSizeBias(float scale) {
  return clamp(1.15 + scale * 0.013, 1.15, 1.70);
}

float layerCentre(vec3 base, float scale, float baseRadius, float density) {
  return craterHeight(base, scale, baseRadius, density, layerSizeBias(scale));
}

float layerGradient(vec3 base, vec3 axis, float scale, float baseRadius, float density) {
  float bias = layerSizeBias(scale);
  float epsilon = baseRadius * 0.45;

  float nearSample = craterHeight(normalize(base + axis * epsilon * 0.5), scale, baseRadius, density, bias);
  float farSample = craterHeight(normalize(base + axis * epsilon * 1.5), scale, baseRadius, density, bias);

  return farSample - nearSample;
}

void main() {
  vec3 geometricNormal = normalize(vWorldNormal);
  vec3 lightDirection = normalize(uLightDirection);
  vec3 viewDirection = normalize(cameraPosition - vWorldPosition);

  vec3 bumpTangent = normalize(vBumpTangent);
  vec3 bumpBitangent = normalize(vBumpBitangent);
  vec3 bumpDirection = normalize(vWorldPosition);

  float largeCentre = layerCentre(bumpDirection, CRATER_LAYER_LARGE_SCALE, CRATER_LAYER_LARGE_RADIUS, CRATER_LARGE_DENSITY);
  float mediumCentre = layerCentre(bumpDirection, CRATER_LAYER_MEDIUM_SCALE, CRATER_LAYER_MEDIUM_RADIUS, CRATER_MEDIUM_DENSITY);
  float fineCentre = layerCentre(bumpDirection, CRATER_LAYER_FINE_SCALE, CRATER_LAYER_FINE_RADIUS, CRATER_FINE_DENSITY);
  float largeTangent = layerGradient(bumpDirection, bumpTangent, CRATER_LAYER_LARGE_SCALE, CRATER_LAYER_LARGE_RADIUS, CRATER_LARGE_DENSITY);
  float mediumTangent = layerGradient(bumpDirection, bumpTangent, CRATER_LAYER_MEDIUM_SCALE, CRATER_LAYER_MEDIUM_RADIUS, CRATER_MEDIUM_DENSITY);
  float fineTangent = layerGradient(bumpDirection, bumpTangent, CRATER_LAYER_FINE_SCALE, CRATER_LAYER_FINE_RADIUS, CRATER_FINE_DENSITY);
  float largeBitangent = layerGradient(bumpDirection, bumpBitangent, CRATER_LAYER_LARGE_SCALE, CRATER_LAYER_LARGE_RADIUS, CRATER_LARGE_DENSITY);
  float mediumBitangent = layerGradient(bumpDirection, bumpBitangent, CRATER_LAYER_MEDIUM_SCALE, CRATER_LAYER_MEDIUM_RADIUS, CRATER_MEDIUM_DENSITY);
  float fineBitangent = layerGradient(bumpDirection, bumpBitangent, CRATER_LAYER_FINE_SCALE, CRATER_LAYER_FINE_RADIUS, CRATER_FINE_DENSITY);

  float bumpAtTangent = tanh((largeTangent + mediumTangent + fineTangent) * LAYER_BUMP_AMPLITUDE * BUMP_GAIN);
  float bumpAtBitangent = tanh((largeBitangent + mediumBitangent + fineBitangent) * LAYER_BUMP_AMPLITUDE * BUMP_GAIN);
  float microBump = (surfaceRoughness(bumpDirection + bumpTangent * MICRO_SAMPLE_DELTA) - surfaceRoughness(bumpDirection)) * MICRO_BUMP_AMPLITUDE;

  vec3 surfaceNormal = normalize(
    geometricNormal - (bumpTangent * (bumpAtTangent + microBump) + bumpBitangent * bumpAtBitangent) * BUMP_STRENGTH
  );

  float crater = sharpenField(largeCentre * 0.34 + mediumCentre * 0.40 + fineCentre * 0.26);
  float maria = mariaField(bumpDirection);

  vec3 highlandBase = mix(uSurfaceColor, uHighlandColor, 0.32);
  vec3 mariaBase = mix(uSurfaceColor, uMariaColor, 0.72);

  vec3 albedo = mix(highlandBase, mariaBase, maria * MARIA_DARKENING + maria * maria * MARIA_DARKENING);
  albedo = mix(albedo, uCraterColor, max(-crater, 0.0) * CRATER_DARKENING);
  albedo = mix(albedo, uHighlandColor, max(crater, 0.0) * CRATER_BRIGHTENING);

  float roughness = mix(0.90, 0.74, maria);
  roughness += crater * 0.06;
  roughness = clamp(roughness, 0.68, 1.0);

  float NdotL = dot(surfaceNormal, lightDirection);
  float terminator = smoothstep(-TERMINATOR_SOFTNESS, TERMINATOR_SOFTNESS, NdotL);
  float directLight = clamp(NdotL, 0.0, 1.0);

  float daylight = uAmbientStrength + (1.0 - uAmbientStrength) * directLight;
  float craterFloor = max(-crater, 0.0);
  float craterWall = smoothstep(0.0, 1.0, abs(crater)) * (1.0 - smoothstep(0.0, 0.9, craterFloor));
  daylight *= 1.0 - craterFloor * uDepthStrength * CRATER_FLOOR_SHADING;
  daylight *= 1.0 - craterWall * uDepthStrength * 0.18;
  vec3 surface = albedo * daylight;

  float NdotV = max(dot(surfaceNormal, viewDirection), 0.0);
  vec3 halfVector = normalize(lightDirection + viewDirection);
  float NdotH = max(dot(surfaceNormal, halfVector), 0.0);
  float shininess = 8.0 + 20.0 * (1.0 - roughness);

  surface += vec3(SPECULAR_STRENGTH * pow(NdotH, shininess) * terminator * (1.0 - maria * 0.8));

  float limbGlow = pow(1.0 - NdotV, 4.0) * LIMB_GLOW_STRENGTH * (1.0 - maria * 0.5);
  surface += vec3(1.0, 0.985, 0.965) * limbGlow;

  surface += uShadowColor * 0.12 * (1.0 - terminator) * (1.0 - maria * 0.4);

  surface = clamp(surface, 0.0, 1.0);
  surface = pow(surface, vec3(1.0 / 2.2));

  gl_FragColor = vec4(surface, 1.0);
}
`
