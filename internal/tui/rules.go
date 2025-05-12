package tui

// GetRulesCompliance returns information about Kubernetes rules compliance
func GetRulesCompliance() string {
	// This would be replaced with actual rules checking logic
	return `✅ Resource Limits: Set correctly
✅ Liveness Probe: Configured
✅ Readiness Probe: Configured
❌ Security Context: Not set
✅ Network Policies: Applied
✅ RBAC: Properly configured`
}
