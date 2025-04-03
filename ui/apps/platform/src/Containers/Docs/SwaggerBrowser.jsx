import React, { useEffect, useState } from 'react';
import { RedocStandalone } from 'redoc';
import Raven from 'raven-js';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import axios from 'services/instance';

export default function SwaggerBrowser({ uri }) {
    const [result, setResult] = useState(null);
    const [loading, setLoading] = useState(true);
    const [isError, setIsError] = useState(false);
    useEffect(() => {
        const fetchData = async () => {
            try {
                const fetchResult = await axios(uri);
                setResult(fetchResult);
                setLoading(false);
            } catch (e) {
                Raven.captureException(e);
                setIsError(true);
            }
        };
        fetchData();
    }, [uri]);
    if (isError) {
        return <div>Unable to load API data.</div>;
    }
    if (loading) {
        return <LoadingSection variant="dark" />;
    }
    if (result) {
        return (
            // Redoc components unreadable with classic dark theme, their styles need to be tuned
            <RedocStandalone spec={result.data} />
        );
    }
}
