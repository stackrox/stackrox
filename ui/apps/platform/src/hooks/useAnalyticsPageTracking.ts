import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

import useAnalytics from './useAnalytics';

function useAnalyticsPageTracking(pageName = '') {
    const { pathname } = useLocation();
    const { analyticsPageVisit } = useAnalytics();

    useEffect(() => {
        analyticsPageVisit('Page Viewed', pageName, { path: pathname });
    }, [pathname, analyticsPageVisit, pageName]);
}

export default useAnalyticsPageTracking;
