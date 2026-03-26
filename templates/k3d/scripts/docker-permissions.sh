#!/bin/bash
# =============================================================================
# Fix Docker socket permissions for dev container
# After restarts, the docker socket permissions may not be set correctly
# =============================================================================

set_docker_permissions() {
    echo "Setting Docker socket permissions..."
    if [ -S /var/run/docker.sock ]; then
        sudo chgrp docker /var/run/docker.sock
        echo "Docker socket permissions set"
    else
        echo "Docker socket not found, skipping permission fix"
    fi
}
