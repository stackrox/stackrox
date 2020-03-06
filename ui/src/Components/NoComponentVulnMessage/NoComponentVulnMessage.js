import React from 'react';

const NoComponentVulnMessage = () => (
    <div className="p-3 leading-normal flex w-full justify-center items-center">
        <div className="p-3 rounded w-full h-full flex items-center justify-center">
            <div>
                <p className="mb-3">
                    No components or vulnerabilities found in this image. This can happen when:
                </p>

                <ul className="list-disc ml-2 pl-2 text-sm text-base-500">
                    <li className="mb-2">
                        Custom binaries or other files are added to the image without using a
                        package manager
                    </li>
                    <li className="mb-2">
                        Packages are added to the image using an unsupported package manager
                    </li>
                    <li>Certain important metadata files are removed from the image</li>
                </ul>
            </div>
        </div>
    </div>
);

export default NoComponentVulnMessage;
