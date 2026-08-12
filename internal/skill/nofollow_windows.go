//go:build windows

package skill

// semSeguirLink não tem equivalente no Windows. A proteção contra link
// simbólico plantado no destino fica por conta do os.Remove seguido de O_EXCL,
// que já não segue link.
const semSeguirLink = 0
