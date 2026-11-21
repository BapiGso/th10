//go:build js && !th10web

package assets

import _ "embed"

const TextureVariant = "web"

//go:embed textures/web.tpk
var texturePack []byte
