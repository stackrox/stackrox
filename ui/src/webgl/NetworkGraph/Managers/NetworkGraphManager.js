import SceneManager from './SceneManager';
import DataManager from './DataManager';

const NetworkGraphManager = element => {
    let networkGraphCanvas;
    let sceneManager;
    let dataManager;
    let onNodeClick;

    let shouldUpdate = true;

    function createCanvas(container) {
        const canvas = document.createElement('canvas');
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        container.appendChild(canvas);
        return canvas;
    }

    // event listeners

    function onClick({ layerX: x, layerY: y }) {
        const node = sceneManager.getNodeAtPosition(x, y);
        if (node) onNodeClick(node);
    }

    function mouseMove({ layerX: x, layerY: y }) {
        sceneManager.onMouseMove(x, y);
    }

    function zoomIn() {
        sceneManager.zoomIn();
    }

    function zoomOut() {
        sceneManager.zoomOut();
    }

    function bindEventListeners() {
        networkGraphCanvas.addEventListener('click', onClick, false);
        networkGraphCanvas.addEventListener('mousemove', mouseMove, false);
    }

    function unbindEventListeners() {
        networkGraphCanvas.removeEventListener('click', onClick, false);
        networkGraphCanvas.removeEventListener('mousemove', mouseMove, false);
    }

    function setUpNetworkData({ nodes, links }) {
        dataManager.setData({ nodes, links });
        const data = dataManager.getData();
        sceneManager.setData(data);
        shouldUpdate = true;
    }

    function setOnNodeClick(callback) {
        onNodeClick = callback;
    }

    function render() {
        requestAnimationFrame(render);
        if (shouldUpdate) {
            sceneManager.update();
            shouldUpdate = false;
        }
        sceneManager.render();
    }

    function setUp() {
        networkGraphCanvas = createCanvas(element);
        dataManager = new DataManager(networkGraphCanvas);
        sceneManager = new SceneManager(networkGraphCanvas);

        bindEventListeners();
        render();
    }

    setUp();

    return {
        unbindEventListeners,
        zoomIn,
        zoomOut,
        setUpNetworkData,
        setOnNodeClick
    };
};

export default NetworkGraphManager;
