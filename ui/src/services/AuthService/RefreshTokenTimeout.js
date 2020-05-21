export default class RefreshTokenTimeout {
    constructor() {
        this.timeoutID = null;
    }

    set(func, delay) {
        this.clear();
        this.timeoutID = setTimeout(func, delay);
    }

    clear() {
        if (this.timeoutID) {
            clearTimeout(this.timeoutID);
            this.timeoutID = null;
        }
    }
}
