import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Slider from 'react-slick';
import * as Icon from 'react-feather';

import SliderButton from 'Components/SliderButton';
import { ClipLoader } from 'react-spinners';

class PanelSlider extends Component {
    static defaultProps = {
        header: '',
        className: 'flex flex-col bg-white border border-base-300',
        children: [],
        onClose: null,
        disablePrevious: false,
        disableNext: false,
        disableFinish: false,
        onNext: null,
        onPrev: null
    };

    static propTypes = {
        header: PropTypes.string,
        className: PropTypes.string,
        children: PropTypes.node,
        onFinish: PropTypes.func.isRequired,
        onClose: PropTypes.func,
        disablePrevious: PropTypes.bool,
        disableNext: PropTypes.bool,
        disableFinish: PropTypes.bool,
        onNext: PropTypes.func,
        onPrev: PropTypes.func
    };

    constructor(props) {
        super(props);

        this.state = {
            index: 0,
            waitForNext: false,
            waitForPrevious: false
        };
    }

    beforeChange = (oldIndex, newIndex) => this.setState({ index: newIndex });

    afterChange = () => this.setState({ waitForNext: false, waitForPrevious: false });

    next = () => {
        if (this.props.onNext) {
            this.setState({ waitForNext: true });
            const promise = this.props.onNext(this.state.index);
            promise
                .then(() => {
                    this.setState({ waitForNext: false });
                    this.slider.slickNext();
                })
                .catch(() => {
                    this.setState({ waitForNext: false });
                });
        } else {
            this.slider.slickNext();
        }
    };

    previous = () => {
        if (this.props.onPrev) {
            this.setState({ waitForPrevious: true });
            const promise = this.props.onNext(this.state.index);
            promise.then(
                () => {
                    this.setState({ waitForPrevious: false });
                    this.slider.slickPrev();
                },
                () => {
                    this.setState({ waitForPrevious: false });
                }
            );
        } else {
            this.slider.slickPrev();
        }
    };

    finish = () => this.props.onFinish();

    renderPreviousButton = () => {
        if (this.state.index !== 0) {
            if (this.state.waitForPrevious) {
                return (
                    <SliderButton type="prev" disabled>
                        <ClipLoader color="currentColor" loading size={14} />
                    </SliderButton>
                );
            }
            return (
                <SliderButton
                    type="prev"
                    onClick={this.previous}
                    disabled={this.props.disablePrevious}
                >
                    Previous
                </SliderButton>
            );
        }
        return '';
    };

    renderNextButton = () => {
        const children = React.Children.toArray(this.props.children);
        if (this.state.index !== children.length - 1) {
            if (this.state.waitForNext) {
                return (
                    <SliderButton type="next" disabled>
                        <ClipLoader color="currentColor" loading size={14} />
                    </SliderButton>
                );
            }
            return (
                <SliderButton type="next" onClick={this.next} disabled={this.props.disableNext}>
                    Next
                </SliderButton>
            );
        }
        return '';
    };

    renderFinishButton = () => {
        const children = React.Children.toArray(this.props.children);
        if (this.state.index === children.length - 1) {
            return (
                <SliderButton
                    type="finish"
                    onClick={this.finish}
                    disabled={this.props.disableFinish}
                >
                    Finish
                </SliderButton>
            );
        }
        return '';
    };

    renderCancelButton() {
        if (!this.props.onClose) return '';
        return (
            <span>
                <button
                    className="cancel flex text-primary-600 p-4 text-center text-sm items-center hover:text-white"
                    onClick={this.props.onClose}
                    data-tip
                    data-for="button-cancel"
                >
                    <Icon.X className="h-4 w-4" />
                </button>
            </span>
        );
    }

    renderButtons() {
        return (
            <div className="flex flex-row">
                {this.renderPreviousButton()}
                {this.renderNextButton()}
                {this.renderFinishButton()}
            </div>
        );
    }

    render() {
        const settings = {
            dots: false,
            infinite: false,
            speed: 500,
            slidesToShow: 1,
            slidesToScroll: 1,
            arrows: false,
            centerMode: true,
            className: '',
            centerPadding: '0px',
            beforeChange: this.beforeChange,
            accessibility: false,
            draggable: false
        };
        return (
            <div className={`${this.props.className}`}>
                <div className="shadow-underline font-bold bg-primary-100">
                    <div className="flex flex-row w-full">
                        <div className="flex flex-1 text-base-600 uppercase items-center tracking-wide py-2 px-4">
                            {this.props.header}
                        </div>
                        <div className="flex items-center py-2 px-4">{this.renderButtons()}</div>
                        <div
                            className={`flex items-end border-base-300 items-center hover:bg-primary-300
                            ${this.props.onClose ? 'ml-2 border-l' : ''}`}
                        >
                            {this.renderCancelButton()}
                        </div>
                    </div>
                </div>
                <div className="h-full overflow-auto">
                    <Slider
                        {...settings}
                        ref={c => {
                            this.slider = c;
                        }}
                    >
                        {this.props.children}
                    </Slider>
                </div>
            </div>
        );
    }
}

export default PanelSlider;
