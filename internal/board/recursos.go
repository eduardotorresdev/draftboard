package board

import _ "embed"

// O estilo e o roteiro são embutidos no binário e escritos inline no HTML: a
// Prancheta tem que abrir por file://, sem servidor, sem rede e sem nenhum
// arquivo ao lado. É a mesma razão pela qual a skill é embutida.

//go:embed prancheta.css
var estilo string

//go:embed prancheta.js
var roteiro string
