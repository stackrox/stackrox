import { useState } from 'react';

function useGraphHoverHint() {
    const [hintData, setHintData] = useState();
    const [hintXY, setHintXY] = useState({});
    const offset = 10;

    function onValueMouseOver(datum) {
        setHintData(datum.hint);
    }

    function onValueMouseOut() {
        setHintData(null);
    }

    function onMouseMove(ev) {
        const container = ev.target.closest('.relative').getBoundingClientRect();
        setHintXY({
            x: ev.clientX - container.left + offset,
            y: ev.clientY - container.top + offset
        });
    }

    const hint = {
        x: hintXY.x,
        y: hintXY.y,
        data: hintData
    };
    return { hint, onValueMouseOver, onValueMouseOut, onMouseMove };
}

export default useGraphHoverHint;
