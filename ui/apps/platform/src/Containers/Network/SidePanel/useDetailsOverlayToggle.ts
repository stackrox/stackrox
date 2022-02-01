import useLocalStorage from 'hooks/useLocalStorage';

type UseDetailsOverlayToggleResult = {
    isContentHidden: boolean;
    toggleContentHidden: () => void;
};

function useDetailsOverlayToggle(): UseDetailsOverlayToggleResult {
    const [isContentHidden, setIsContentHidden] = useLocalStorage(
        'networkDetailOverlayToggleValue',
        false
    );

    function toggleContentHidden() {
        setIsContentHidden(!isContentHidden);
    }

    return { isContentHidden, toggleContentHidden };
}

export default useDetailsOverlayToggle;
