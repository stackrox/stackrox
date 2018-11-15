import React from 'react';
import SwaggerBrowserComponent from './SwaggerBrowserComponent';

function ApiPage() {
    return <SwaggerBrowserComponent uri="/api/docs/swagger" />;
}

export default ApiPage;
