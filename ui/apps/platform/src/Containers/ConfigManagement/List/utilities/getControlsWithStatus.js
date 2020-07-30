import { standardLabels } from 'messages/standards';
import COMPLIANCE_STATES from 'constants/complianceStates';

const getControlStatus = (state) => {
    let status = null;
    if (state === 'COMPLIANCE_STATE_FAILURE') {
        status = COMPLIANCE_STATES.FAIL;
    } else if (state === 'COMPLIANCE_STATE_SUCCESS') {
        status = COMPLIANCE_STATES.PASS;
    } else {
        status = COMPLIANCE_STATES['N/A'];
    }
    return status;
};

export default function getControlsWithStatus(data) {
    const controls = {};
    data.forEach(({ control, value }) => {
        if (controls[control.id] && controls[control.id].status !== COMPLIANCE_STATES.FAIL) {
            controls[control.id].status = getControlStatus(value.overallState);
        } else if (!controls[control.id]) {
            const controlObj = { ...control };
            controlObj.standard = standardLabels[control.standardId];
            controlObj.control = `${control.name} - ${control.description}`;
            controlObj.status = getControlStatus(value.overallState);
            controls[control.id] = controlObj;
        }
    });
    return Object.values(controls);
}
