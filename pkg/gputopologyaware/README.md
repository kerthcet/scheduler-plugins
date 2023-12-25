# GPU Topology-Aware Scheduling Plugin

## Note

GPUTopologyAware should be configured after NodeResource plugin

// Note:
// This is mostly inspired by <https://github.com/NVIDIA/go-gpuallocator>.
// However, this library can not be used directly for the limitations of
// designed for device plugin only.
// We raised issues here hope to extend the scope of this library:
// <https://github.com/NVIDIA/go-gpuallocator/issues/19>.
