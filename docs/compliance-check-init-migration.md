# Compliance Check Init Migration

## Current State
- 109 compliance check files have init() functions
- Each calls framework.MustRegisterChecks() individually
- Registration happens via package init when imported

## Migration Strategy
Phase 3.2 establishes initCompliance() stub in central/app/init.go

Future work (separate PR):
1. Refactor pkg/compliance/checks to export registration functions
2. Call those functions from central/app/init.go initCompliance()
3. Remove init() from all 109 check files

## Files affected:

### central/compliance/checks/
- remote/all.go
- nist80053/check_cm_11/check.go
- nist80053/check_cm_7/check.go
- nist80053/check_sa_10/check.go
- nist80053/check_ra_3/check.go
- nist80053/check_ac_14/check.go
- nist80053/check_cm_3/check.go
- nist80053/check_cm_2/check.go
- nist80053/check_si_4/check.go
- nist80053/check_ir_4_5/check.go
- nist80053/check_cm_8/check.go
- nist80053/check_sc_6/check.go
- nist80053/check_ir_5/check.go
- nist80053/check_cm_6/check.go
- nist80053/check_ir_6_1/check.go
- nist80053/check_ca_9/check.go
- nist80053/check_cm_5/check.go
- nist80053/check_sc_7/check.go
- nist80053/check_ra_5/check.go
- nist80053/check_si_3_8/check.go
- nist80053/check_si_2_2/check.go
- pcidss32/check811/check.go
- pcidss32/check12/check.go
- pcidss32/check24/check.go
- pcidss32/check71/check.go
- pcidss32/check62/check.go
- pcidss32/check85/check.go
- pcidss32/check22/check.go
- pcidss32/check362/check.go
- pcidss32/check712/check.go
- pcidss32/check112/check.go
- pcidss32/check225/check.go
- pcidss32/check134/check.go
- pcidss32/check722/check.go
- pcidss32/check723/check.go
- pcidss32/check21/check.go
- pcidss32/check23/check.go
- pcidss32/check135/check.go
- pcidss32/check132/check.go
- pcidss32/check61/check.go
- pcidss32/check1121/check.go
- pcidss32/check114/check.go
- pcidss32/check121/check.go
- pcidss32/check656/check.go
- pcidss32/check713/check.go
- nist800-190/check435/check.go
- nist800-190/check411/check.go
- nist800-190/check412/check.go
- nist800-190/check433/check.go
- nist800-190/check442/check.go
- nist800-190/check414/check.go
- nist800-190/check444/check.go
- nist800-190/check432/check.go
- nist800-190/check422/check.go
- nist800-190/check455/check.go
- nist800-190/check451/check.go
- nist800-190/check431/check.go
- nist800-190/check443/check.go
- hipaa_164/check312e/check.go
- hipaa_164/check312c/check.go
- hipaa_164/check308a7iie/check.go
- hipaa_164/check306e/check.go
- hipaa_164/check312e1/check.go
- hipaa_164/check316b2iii/check.go
- hipaa_164/check308a1i/check.go
- hipaa_164/check308a1iia/check.go
- hipaa_164/check308a5iib/check.go
- hipaa_164/check308a1iib/check.go
- hipaa_164/check310a1/check.go
- hipaa_164/check308a6ii/check.go
- hipaa_164/check308a3iib/check.go
- hipaa_164/check308a4iib/check.go
- hipaa_164/check310a1a/check.go
- hipaa_164/check308a3iia/check.go
- hipaa_164/check314a2ic/check.go
- hipaa_164/check308a4/check.go
- hipaa_164/check310d/check.go

### pkg/compliance/checks/
- nist80053/check_ac_14/check.go
- nist80053/check_ac_3_7/check.go
- nist80053/check_cm_5/check.go
- nist80053/check_ac_24/check.go
- pcidss32/check811/check.go
- pcidss32/check71/check.go
- pcidss32/check85/check.go
- pcidss32/check362/check.go
- pcidss32/check712/check.go
- pcidss32/check722/check.go
- pcidss32/check723/check.go
- pcidss32/check713/check.go
- nist800-190/check421/check.go
- nist800-190/check432/check.go
- nist800-190/check431/check.go
- hipaa_164/check312e1/check.go
- hipaa_164/check308a3iib/check.go
- hipaa_164/check308a4/check.go
- kubernetes/master_scheduler.go
- kubernetes/policies_network_cni.go
- kubernetes/policies_secrets_management.go
- kubernetes/kubelet_command.go
- kubernetes/worker_node_config.go
- kubernetes/policies_admission_control.go
- kubernetes/control_plane_config.go
- kubernetes/master_api_server.go
- kubernetes/policies_rbac.go
- kubernetes/policies_pod_security.go
- kubernetes/master_config.go
- kubernetes/etcd.go
- kubernetes/master_controller_manager.go
- kubernetes/policies_general.go
