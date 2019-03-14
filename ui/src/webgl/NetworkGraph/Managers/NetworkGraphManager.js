import throttle from 'lodash/throttle';
import debounce from 'lodash/debounce';

import SceneManager from './SceneManager';
import DataManager from './DataManager';

const THROTTLE_DELAY = 500;
const DEBOUNCE_DELAY = 15;

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

    function render() {
        if (shouldUpdate) {
            sceneManager.update();
            shouldUpdate = false;
        }
        sceneManager.render();
    }

    // event listeners

    function onClick({ layerX: x, layerY: y }) {
        const node = sceneManager.getNodeAtPosition(x, y);
        if (node) onNodeClick(node);
        render();
    }

    function onMouseMove({ layerX: x, layerY: y }) {
        sceneManager.onMouseMove(x, y);
        render();
    }

    function onMouseWheel() {
        render();
    }

    function zoomIn() {
        sceneManager.zoomIn();
    }

    function zoomOut() {
        sceneManager.zoomOut();
    }

    const onThrottleClick = throttle(onClick, THROTTLE_DELAY, { trailing: false });

    const onDebounceMouseMove = debounce(onMouseMove, DEBOUNCE_DELAY);

    function bindEventListeners() {
        networkGraphCanvas.addEventListener('click', onThrottleClick, false);
        networkGraphCanvas.addEventListener('mousemove', onDebounceMouseMove, false);
        networkGraphCanvas.addEventListener('wheel', onMouseWheel, false);
    }

    function unbindEventListeners() {
        networkGraphCanvas.removeEventListener('click', onThrottleClick, false);
        networkGraphCanvas.removeEventListener('mousemove', onDebounceMouseMove, false);
        networkGraphCanvas.addEventListener('wheel', onMouseWheel, false);
    }

    function setUpNetworkData({ nodes, networkFlowMapping, worker }) {
        dataManager.setData({ nodes, networkFlowMapping, worker });
    }

    function setNetworkNodes(nodes) {
        dataManager.setNodes(nodes);
    }

    function setNetworkLinks(links) {
        dataManager.setLinks(links);
    }

    function setNetworkNamespaces(namespaces) {
        dataManager.setNamespaces(namespaces);
    }

    function renderNetworkGraph() {
        const data = dataManager.getData();
        sceneManager.setData(data);
        shouldUpdate = true;
        render();
    }

    function setOnNodeClick(callback) {
        onNodeClick = callback;
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
        dataManager,
        unbindEventListeners,
        zoomIn,
        zoomOut,
        setUpNetworkData,
        renderNetworkGraph,
        setOnNodeClick,
        setNetworkNodes,
        setNetworkLinks,
        setNetworkNamespaces
    };
};

export default NetworkGraphManager;
