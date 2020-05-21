/* eslint-disable func-names */
// workaround to get cypress to interact with react-dnd
// https://github.com/cypress-io/cypress/issues/1752

function DndSimulatorDataTransfer() {
    this.data = {};
}
DndSimulatorDataTransfer.prototype.dropEffect = 'move';
DndSimulatorDataTransfer.prototype.effectAllowed = 'all';
DndSimulatorDataTransfer.prototype.files = [];
DndSimulatorDataTransfer.prototype.items = [];
DndSimulatorDataTransfer.prototype.types = [];

DndSimulatorDataTransfer.prototype.clearData = function (format) {
    if (format) {
        delete this.data[format];

        const index = this.types.indexOf(format);
        delete this.types[index];
        delete this.data[index];
    } else {
        this.data = {};
    }
};

DndSimulatorDataTransfer.prototype.setData = function (format, data) {
    this.data[format] = data;
    this.items.push(data);
    this.types.push(format);
};

DndSimulatorDataTransfer.prototype.getData = function (format) {
    if (format in this.data) {
        return this.data[format];
    }

    return '';
};

DndSimulatorDataTransfer.prototype.setDragImage = function () {
    // since simulation doesn"t replicate the visual
    // effects, there is no point in implementing this
};

export default DndSimulatorDataTransfer;
