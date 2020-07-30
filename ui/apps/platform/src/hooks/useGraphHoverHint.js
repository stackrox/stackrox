import { useState } from 'react';

function useGraphHoverHint() {
    const [hint, setHint] = useState();

    function onValueMouseOver(datum, { event }) {
        if (datum.hint) {
            setHint({ data: datum.hint, target: event.target });
        }
    }

    function onValueMouseOut() {
        setHint(null);
    }

    return { hint, onValueMouseOver, onValueMouseOut };
}

export default useGraphHoverHint;
