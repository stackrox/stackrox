import { useEffect, useState } from 'react';
import { RedocStandalone } from 'redoc';
import Raven from 'raven-js';
import type { AxiosResponse } from 'axios';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import axios from 'services/instance';

export type SwaggerBrowserProps = {
    uri: string;
};

function SwaggerBrowser({ uri }: SwaggerBrowserProps) {
    const [result, setResult] = useState<AxiosResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [isError, setIsError] = useState(false);

    useEffect(() => {
        axios(uri)
            .then((fetchResult) => {
                setResult(fetchResult);
            })
            .catch((e) => {
                Raven.captureException(e);
                setIsError(true);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [uri]);

    if (isError) {
        return <div>Unable to load API data.</div>;
    }
    if (loading) {
        return <LoadingSection variant="dark" />;
    }
    if (result?.data) {
        return (
            // Redoc components unreadable with classic dark theme, their styles need to be tuned
            <RedocStandalone spec={result.data} />
        );
    }
}

export default SwaggerBrowser;
