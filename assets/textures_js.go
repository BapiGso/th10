//go:build js

package assets

import _ "embed"

const TextureVariant = "web"

//go:embed textures/web.tpk
var texturePack []byte
