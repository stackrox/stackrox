import React from 'react';
import SwaggerBrowser from './SwaggerBrowser';

function ApiPage() {
    return <SwaggerBrowser uri="/api/docs/swagger" />;
}

export default ApiPage;
