package ingestion

// This is the pipeline for event ingestion.
// This is a sensor wide component that listens to kubernetes and collector event inputs.
// It has three internal components that run in parallel:
//   - K8s Event Handlers
//   - Internal hash-map stores (holds all pre-computed resources)
//   - Resource Dependency Graph (holds all relationship between resources)
//   - "Big Momma" the processing engine that creates update snapshots
//   - Object enhancer
//   - Detector engine


