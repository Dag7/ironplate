package plugin

// DefaultRegistry is the global plugin registry with all built-in plugins.
var DefaultRegistry = NewRegistry()

// RegisterDefaults registers all built-in plugins.
// Called at startup from the root command.
func RegisterDefaults(r *Registry) error {
	// Cloud providers will be registered here as they are implemented.
	// Example: r.RegisterCloudProvider(gcp.New())

	// Service generators will be registered here.
	// Example: r.RegisterServiceGenerator(node.NewAPIGenerator())

	// Infrastructure components will be registered here.
	// Example: r.RegisterInfraComponent(kafka.New())

	return nil
}
