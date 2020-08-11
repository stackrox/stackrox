import React, { useEffect, useState } from 'react';
import { RedocStandalone } from 'redoc';
import Raven from 'raven-js';

import LoadingSection from 'Components/LoadingSection';
import axios from 'services/instance';

function SwaggerBrowser({ uri }) {
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
        return <LoadingSection />;
    }
    if (result) {
        return (
            // Redoc components unreadable with StackRox dark theme, their styles need to be tuned
            // for now just always enforcing light theme
            <div className="theme-light">
                <RedocStandalone spec={result.data} />
            </div>
        );
    }
}

function ApiPage() {
    return <SwaggerBrowser uri="/api/docs/swagger" />;
}

export default ApiPage;
