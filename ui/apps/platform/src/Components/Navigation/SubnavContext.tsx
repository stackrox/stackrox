import { createContext, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

type SubnavContextValue = {
    content: ReactNode;
    setContent: (content: ReactNode) => void;
};

const SubnavContext = createContext<SubnavContextValue>({
    content: null,
    setContent: () => {},
});

export function SubnavProvider({ children }: { children: ReactNode }) {
    const [content, setContent] = useState<ReactNode>(null);

    const value = useMemo(() => ({ content, setContent }), [content]);

    return <SubnavContext.Provider value={value}>{children}</SubnavContext.Provider>;
}

export function Subnav({ children }: { children: ReactNode }) {
    const { setContent } = useContext(SubnavContext);

    useEffect(() => {
        setContent(children);
        return () => setContent(null);
    }, [children, setContent]);

    return null;
}

export function useSubnavContent() {
    return useContext(SubnavContext);
}
