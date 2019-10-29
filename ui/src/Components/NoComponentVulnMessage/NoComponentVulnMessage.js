import React from 'react';

const NoComponentVulnMessage = () => (
    <div className="p-3">
        <div className="pl-3">No components or vulnerabilities found in this image.</div>
        <div className="pl-3 pt-3">This can happen when:</div>
        <div className="pl-3 pt-1">
            <ul className="pl-4">
                <li className="pt-1">
                    custom binaries or other files are added to the image without using a package
                    manager;
                </li>
                <li className="pt-1">
                    packages are added to the image using an unsupported package manager; or
                </li>
                <li className="pt-1">
                    certain important metadata files are removed from the image.
                </li>
            </ul>
        </div>
    </div>
);

export default NoComponentVulnMessage;
