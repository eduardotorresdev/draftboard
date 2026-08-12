//go:build !windows

package skill

import "syscall"

// semSeguirLink impede que a criação do arquivo siga um link simbólico no
// caminho final. Fecha a janela de corrida entre o os.Remove e o open.
const semSeguirLink = syscall.O_NOFOLLOW
