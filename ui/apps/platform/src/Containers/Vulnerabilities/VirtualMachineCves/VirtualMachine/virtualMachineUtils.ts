import type { AgentStatus, VirtualMachineV2State } from 'services/VirtualMachineService';

export const stateDisplayMap: Record<VirtualMachineV2State, string> = {
    VM_STATE_UNKNOWN: 'Unknown',
    VM_STATE_STOPPED: 'Stopped',
    VM_STATE_RUNNING: 'Running',
};

export const agentStatusDisplayMap: Record<AgentStatus, string> = {
    AGENT_STATUS_UNKNOWN: 'Unknown',
    AGENT_STATUS_ACTIVE: 'Active',
    AGENT_STATUS_INACTIVE: 'Inactive',
};
