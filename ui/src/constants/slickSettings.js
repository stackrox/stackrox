import React from 'react';
import { CarouselNextArrow, CarouselPrevArrow } from 'Components/CarouselArrows';

const emptyFunc = () => null;
const slickSettings = {
    dots: false,
    nextArrow: <CarouselNextArrow onClick={emptyFunc} />,
    prevArrow: <CarouselPrevArrow onClick={emptyFunc} />
};

export default slickSettings;
