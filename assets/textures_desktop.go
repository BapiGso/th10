//go:build !js

package assets

import _ "embed"

// TextureVariant identifies the selected pixel policy for resource tests.
const TextureVariant = "lossless"

//go:embed textures/lossless.tpk
var texturePack []byte
