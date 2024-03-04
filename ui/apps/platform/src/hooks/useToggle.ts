import { useCallback, useState } from 'react';

type UseToggleReturn = {
    /** Whether or not the toggle is on */
    isOn: boolean;
    /** Callback that fires when the toggle state changes */
    onToggle: (isOn: boolean) => void;
    /** Function that sets the toggle state */
    toggle: (toggleState: boolean) => void;
    /** Function that sets the state to true */
    toggleOn: () => void;
    /** Function that sets the state to false */
    toggleOff: () => void;
};

/**
 * Hook to handle general true/false toggle states
 */
function useToggle(defaultState = false): UseToggleReturn {
    const [isOn, setIsOn] = useState<boolean>(defaultState);
    const onToggle = useCallback(() => setIsOn(!isOn), [isOn, setIsOn]);
    const toggle = useCallback(setIsOn, [setIsOn]);
    const toggleOn = useCallback(() => toggle(true), [toggle]);
    const toggleOff = useCallback(() => toggle(false), [toggle]);

    return { isOn, onToggle, toggle, toggleOn, toggleOff };
}

export default useToggle;
