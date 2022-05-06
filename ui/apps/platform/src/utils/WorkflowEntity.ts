// An item in the workflow stack
export default class WorkflowEntity {
    t: string | null;

    i?: string;

    constructor(entityType: string | null, entityId?: string) {
        this.t = entityType;
        this.i = entityId;
        Object.freeze(this);
    }

    get entityType() {
        return this.t;
    }

    get entityId() {
        return this.i;
    }
}
