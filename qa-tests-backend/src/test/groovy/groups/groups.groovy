package groups

@interface RUNTIME { }
@interface SMOKE { }
@interface COMPATIBILITY { }
@interface BAT { }
@interface Integration { }
@interface NetworkPolicySimulation { }
@interface PolicyEnforcement { }
@interface NetworkFlowVisualization { }
@interface NetworkBaseline { }
@interface Upgrade { }
@interface SensorBounce { }       // First batch of sensor bounce tests that expect no previous bounce
@interface SensorBounceNext { } // Next batch that don't care
@interface GraphQL { }
@interface Notifiers { }
@interface K8sEvents { }
@interface Begin { } // Tests that needs to be run before all other tests
