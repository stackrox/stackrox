import React, { useState, useEffect, useCallback, ReactElement } from 'react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import {
    Card,
    CardBody,
    CardTitle,
    ClipboardCopy,
    Banner,
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    ExpandableSection,
    ExpandableSectionToggle,
    Flex,
    FlexItem,
} from '@patternfly/react-core';
import { ExclamationTriangleIcon, PlayIcon } from '@patternfly/react-icons';
import { AxiosError } from 'axios';

import axios from 'services/instance';

export type ApiDebugInfo = {
    id: string;
    url: string;
    method: string;
    requestData: unknown;
    requestHeaders: Record<string, unknown>;
    responseData: unknown;
    responseHeaders: Record<string, unknown>;
    status: number;
    duration: number;
    timestamp: Date;
    isError: boolean;
    errorMessage?: string;
};

/* -----------------------------------------------------------------------------
 *  Store & helpers
 * -------------------------------------------------------------------------- */

const debugStore: Record<string, ApiDebugInfo> = {};

export const isDebugEnabled = (): boolean => {
    if (typeof window === 'undefined') {
        return false;
    }

    const params = new URLSearchParams(window.location.search);
    const debugParam = params.get('debug') === 'true';
    const debugLocalStorage = localStorage.getItem('apiDebugEnabled') === 'true';

    if (debugParam && !debugLocalStorage) {
        try {
            localStorage.setItem('apiDebugEnabled', 'true');
        } catch {
            // catch
        }
    }

    return debugParam || debugLocalStorage;
};

export const setDebugMode = (enabled: boolean): void => {
    const url = new URL(window.location.href);

    if (enabled) {
        url.searchParams.set('debug', 'true');
        localStorage.setItem('apiDebugEnabled', 'true');
    } else {
        url.searchParams.delete('debug');
        localStorage.removeItem('apiDebugEnabled');
    }

    window.history.replaceState({}, '', url.toString());
    window.dispatchEvent(new CustomEvent('debug-mode-changed', { detail: { enabled } }));
};

/* -----------------------------------------------------------------------------
 *  Axios interceptors
 * -------------------------------------------------------------------------- */

let interceptorsInitialized = false;

export const initializeApiDebugInterceptors = (): void => {
    if (interceptorsInitialized) {
        return;
    }

    axios.interceptors.request.use(
        (incoming) => {
            if (!isDebugEnabled()) {
                return incoming;
            }

            const requestId = `${incoming.url}_${Date.now()}`;
            const startTime = Date.now();

            debugStore[requestId] = {
                id: requestId,
                url: incoming.url || '',
                method: incoming.method?.toUpperCase() || 'GET',
                requestData: incoming.data,
                requestHeaders: incoming.headers,
                responseData: null,
                responseHeaders: {},
                status: 0,
                duration: 0,
                timestamp: new Date(),
                isError: false,
            };

            const requestConfig = { ...incoming };

            // @ts-expect-error attaching metadata to AxiosRequestConfig
            requestConfig.metadata = {
                requestId,
                startTime,
            };

            return requestConfig;
        },
        (error) => {
            return Promise.reject(error);
        }
    );

    axios.interceptors.response.use(
        (response) => {
            if (!isDebugEnabled()) {
                return response;
            }

            // @ts-expect-error pulling metadata from above
            const { requestId, startTime } = response.config.metadata;

            if (requestId && debugStore[requestId]) {
                debugStore[requestId] = {
                    ...debugStore[requestId],
                    responseData: response.data,
                    responseHeaders: response.headers,
                    status: response.status,
                    duration: Date.now() - startTime,
                };

                window.dispatchEvent(
                    new CustomEvent('api-debug-update', {
                        detail: { requestId, info: debugStore[requestId] },
                    })
                );
            }

            return response;
        },
        (error: AxiosError) => {
            if (!isDebugEnabled()) {
                return Promise.reject(error);
            }

            // @ts-expect-error metadata from above
            const { requestId, startTime } = error.config?.metadata ?? {};

            if (requestId && debugStore[requestId]) {
                debugStore[requestId] = {
                    ...debugStore[requestId],
                    responseData: error.response?.data,
                    responseHeaders: error.response?.headers || {},
                    status: error.response?.status || 0,
                    duration: startTime !== undefined ? Date.now() - startTime : 0,
                    isError: true,
                    errorMessage: error.message,
                };

                window.dispatchEvent(
                    new CustomEvent('api-debug-update', {
                        detail: { requestId, info: debugStore[requestId] },
                    })
                );
            }

            return Promise.reject(error);
        }
    );

    interceptorsInitialized = true;
};
/* -----------------------------------------------------------------------------
 *  Hooks
 * -------------------------------------------------------------------------- */

function useApiDebugInfo(urlPattern?: string | RegExp, paused = false): ApiDebugInfo[] {
    const [debugInfo, setDebugInfo] = useState<ApiDebugInfo[]>([]);
    const [enabled, setEnabled] = useState(isDebugEnabled);

    const urlMatches = useCallback(
        (url: string) => {
            if (!urlPattern) {
                return true;
            }
            const path = url.split('?')[0];
            return typeof urlPattern === 'string'
                ? path === urlPattern || url === urlPattern
                : urlPattern.test(url);
        },
        [urlPattern]
    );

    useEffect(() => {
        if (paused) {
            return () => {};
        }

        const handleUpdate = (e: Event): void => {
            const { info } = (e as CustomEvent<{ info: ApiDebugInfo }>).detail;
            if (!urlMatches(info.url)) {
                return;
            }
            setDebugInfo((prev) => {
                const next = [info, ...prev.filter((i) => i.id !== info.id)];
                return next.slice(0, 20);
            });
        };

        const handleModeChange = (): void => setEnabled(isDebugEnabled());

        if (enabled) {
            const initial = Object.values(debugStore)
                .filter((info) => urlMatches(info.url))
                .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
                .slice(0, 20);
            setDebugInfo(initial);
        } else {
            setDebugInfo([]);
        }

        window.addEventListener('api-debug-update', handleUpdate);
        window.addEventListener('debug-mode-changed', handleModeChange);
        return () => {
            window.removeEventListener('api-debug-update', handleUpdate);
            window.removeEventListener('debug-mode-changed', handleModeChange);
        };
    }, [enabled, paused, urlMatches]);

    return enabled ? debugInfo : [];
}

export interface UseApiMockReturn<ReturnType> {
    mockData: ReturnType | undefined;
    DebugPanelComponent: React.FC;
}

export function useApiMock<ReturnType>(urlPattern: string | RegExp): UseApiMockReturn<ReturnType> {
    const [mockData, setMockData] = useState<ReturnType | undefined>(undefined);

    const handleMockResponse = useCallback((payload: ReturnType) => {
        setMockData(payload);
    }, []);

    const DebugPanelComponent = useCallback(() => {
        if (!isDebugEnabled()) {
            return null;
        }
        return <ApiDebugPanel urlPattern={urlPattern} onMockResponse={handleMockResponse} />;
    }, [urlPattern, handleMockResponse]);

    return { mockData, DebugPanelComponent };
}

/* -----------------------------------------------------------------------------
 *  UI components
 * -------------------------------------------------------------------------- */

export const DebugModeBanner = (): ReactElement | null => {
    const [enabled, setEnabled] = useState(isDebugEnabled());

    useEffect(() => {
        const handler = (event: Event): void => {
            const { enabled: val } = (event as CustomEvent<{ enabled: boolean }>).detail;
            setEnabled(val);
        };
        window.addEventListener('debug-mode-changed', handler);
        return () => window.removeEventListener('debug-mode-changed', handler);
    }, []);

    if (!enabled) {
        return null;
    }

    const toggle = (): void => {
        setDebugMode(false);
        setEnabled(false);
        window.location.reload();
    };

    return (
        <Banner
            variant="gold"
            onClick={toggle}
            style={{ cursor: 'pointer' }}
            screenReaderText="Warning banner"
        >
            <Flex
                justifyContent={{ default: 'justifyContentCenter' }}
                spaceItems={{ default: 'spaceItemsSm' }}
            >
                <FlexItem>
                    <ExclamationTriangleIcon />
                </FlexItem>
                <FlexItem>Debug mode enabled – click to disable</FlexItem>
            </Flex>
        </Banner>
    );
};

interface ApiDebugPanelProps<ReturnType> {
    urlPattern?: string | RegExp;
    isReadOnly?: boolean;
    title?: string;
    onMockResponse?: (response: ReturnType) => void;
}

export const ApiDebugPanel = <ReturnType,>({
    urlPattern,
    onMockResponse,
    isReadOnly = false,
    title = 'API Debug Info',
}: ApiDebugPanelProps<ReturnType>): ReactElement | null => {
    const [paused, setPaused] = useState(false);
    const liveInfo = useApiDebugInfo(urlPattern, paused);
    const latest = liveInfo[0];
    const [isExpanded, setExpanded] = useState(false);
    const [code, setCode] = useState<string>('');

    useEffect(() => {
        if (latest) {
            setCode(JSON.stringify(latest.responseData, null, 2));
        }
    }, [latest]);

    if (!latest) {
        return null;
    }

    const execute = (): void => {
        try {
            const parsed = JSON.parse(code);
            setExpanded(false);
            onMockResponse?.(parsed);
        } catch {
            // show error
        }
    };

    return (
        <div style={{ border: '1px solid #F0AB00', margin: '10px', position: 'relative' }}>
            <div style={{ color: '#F0AB00', position: 'absolute', top: 3, right: 5 }}>
                DEBUG PANEL
            </div>
            <Card isCompact isPlain ouiaId="api-debug-card">
                <CardTitle>{title}</CardTitle>
                <CardBody>
                    <DescriptionList isHorizontal isCompact>
                        <DescriptionListGroup>
                            <DescriptionListTerm>URL</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ClipboardCopy variant="inline-compact">{latest.url}</ClipboardCopy>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Time</DescriptionListTerm>
                            <DescriptionListDescription>
                                {latest.timestamp.toLocaleString()}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Duration</DescriptionListTerm>
                            <DescriptionListDescription>
                                {latest.duration} ms
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Response</DescriptionListTerm>
                            <DescriptionListDescription>
                                <ExpandableSectionToggle
                                    isExpanded={isExpanded}
                                    onToggle={(expanded) => setExpanded(expanded)}
                                    toggleId="response-toggle"
                                    contentId="response-toggle"
                                >
                                    {isExpanded ? 'Hide response' : 'Show response'}
                                </ExpandableSectionToggle>
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                    {isExpanded && (
                        <ExpandableSection
                            toggleId="response-toggle"
                            contentId="response-toggle"
                            isDetached
                            isExpanded
                        >
                            <CodeEditor
                                code={code}
                                onChange={setCode}
                                isCopyEnabled
                                isDownloadEnabled
                                isReadOnly={isReadOnly}
                                customControls={
                                    <CodeEditorControl
                                        icon={<PlayIcon />}
                                        aria-label="Mock response"
                                        tooltipProps={{ content: 'Mock response' }}
                                        onClick={execute}
                                        isVisible={!isReadOnly}
                                    />
                                }
                                height="500px"
                                language={Language.json}
                            />
                        </ExpandableSection>
                    )}
                    <Button
                        size="sm"
                        onClick={() => setPaused((paused) => !paused)}
                        style={{ marginTop: 8 }}
                    >
                        {paused ? 'Resume' : 'Pause'}
                    </Button>
                </CardBody>
            </Card>
        </div>
    );
};
