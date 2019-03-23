import React from 'react';
import { stackroxSupport } from 'messages/common';
import * as Icon from 'react-feather';
import Widget from 'Components/Widget';

const UpgradeSupport = () => (
    <Widget header="How To Renew Or Upgrade Your License">
        <div className="flex flex-col flex-1 w-full leading-loose">
            <div className="py-4 px-6 text-base-600 leading-loose">
                To renew or upgrade your StackRox Kubernetes Security Platform license, please
                contact our customer success team over phone or email and we will respond in 24
                hours or less.
            </div>
            <div className="flex flex-1 border-t border-base-400">
                <div className="flex flex-1 items-center justify-center text-center border-r border-base-400">
                    <div>
                        <div className="text-primary-800 font-400 text-4xl">
                            <Icon.Phone className="h-6 w-6 text-primary-800" />
                        </div>
                        <div>
                            <a
                                className="text-base-600 tracking-wide"
                                href={`tel:+${stackroxSupport.phoneNumber.withDashes}`}
                            >
                                {stackroxSupport.phoneNumber.withSpaces}
                            </a>
                        </div>
                    </div>
                </div>
                <div className="flex flex-1 items-center justify-center text-center">
                    <div>
                        <div className="text-primary-800 font-400 text-4xl">
                            <Icon.Mail className="h-6 w-6 text-primary-800" />
                        </div>
                        <div>
                            <a
                                className="text-base-600 tracking-wide"
                                href={`mailto:${
                                    stackroxSupport.email
                                }?subject=StackRox License Renewal&body=I would like to renew my StackRox Kubernetes Security Platform License.`}
                            >
                                {stackroxSupport.email}
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </Widget>
);

export default UpgradeSupport;
