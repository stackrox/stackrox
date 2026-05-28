import { useSearchParams } from 'react-router-dom-v5-compat';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

export type LayoutMode = 'flow' | 'tabs' | 'collapsible';

const LAYOUT_PARAM = 'layout';

/**
 * Returns the current layout mode from the URL query parameter, defaulting to 'flow'.
 */
export function useLayoutMode(): [LayoutMode, (mode: LayoutMode) => void] {
    const [searchParams, setSearchParams] = useSearchParams();
    const raw = searchParams.get(LAYOUT_PARAM);
    const mode: LayoutMode = raw === 'tabs' || raw === 'collapsible' ? raw : 'flow';

    function setMode(next: LayoutMode) {
        setSearchParams((prev) => {
            const updated = new URLSearchParams(prev);
            updated.set(LAYOUT_PARAM, next);
            return updated;
        });
    }

    return [mode, setMode];
}

type LayoutToggleProps = {
    mode: LayoutMode;
    onSelect: (mode: LayoutMode) => void;
};

/**
 * PatternFly ToggleGroup for switching between Flow, Tabs, and Collapsible layouts.
 */
function LayoutToggle({ mode, onSelect }: LayoutToggleProps) {
    return (
        <ToggleGroup aria-label="Layout selector">
            <ToggleGroupItem
                text="Flow"
                buttonId="layout-flow"
                isSelected={mode === 'flow'}
                onChange={() => onSelect('flow')}
            />
            <ToggleGroupItem
                text="Tabs"
                buttonId="layout-tabs"
                isSelected={mode === 'tabs'}
                onChange={() => onSelect('tabs')}
            />
            <ToggleGroupItem
                text="Collapsible"
                buttonId="layout-collapsible"
                isSelected={mode === 'collapsible'}
                onChange={() => onSelect('collapsible')}
            />
        </ToggleGroup>
    );
}

export default LayoutToggle;
