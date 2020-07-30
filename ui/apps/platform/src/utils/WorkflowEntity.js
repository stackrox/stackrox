// An item in the workflow stack
export default class WorkflowEntity {
    constructor(entityType, entityId) {
        if (entityType) {
            this.t = entityType;
        }
        if (entityId) {
            this.i = entityId;
        }
        Object.freeze(this);
    }

    get entityType() {
        return this.t;
    }

    get entityId() {
        return this.i;
    }
}
