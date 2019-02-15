/*

Tailwind - The Utility-First CSS Framework

A project by Adam Wathan (@adamwathan), Jonathan Reinink (@reinink),
David Hemphill (@davidhemphill) and Steve Schoger (@steveschoger).

Welcome to the Tailwind config file. This is where you can customize
Tailwind specifically for your project. Don't be intimidated by the
length of this file. It's really just a big JavaScript object and
we've done our very best to explain each section.

View the full documentation at https://tailwindcss.com.


|-------------------------------------------------------------------------------
| The default config
|-------------------------------------------------------------------------------
|
| This variable contains the default Tailwind config. You don't have
| to use it, but it can sometimes be helpful to have available. For
| example, you may choose to merge your custom configuration
| values with some of the Tailwind defaults.
|
*/

const defaultConfig = require('tailwindcss/defaultConfig')();
const fs = require('fs');

const textBase = '16';

function remCalc(pixel) {
    return `${pixel / textBase}rem`;
}

/*
|-------------------------------------------------------------------------------
| Colors                                    https://tailwindcss.com/docs/colors
|-------------------------------------------------------------------------------
|
| Here you can specify the colors used in your project. To get you started,
| we've provided a generous palette of great looking colors that are perfect
| for prototyping, but don't hesitate to change them for your project. You
| own these colors, nothing will break if you change everything about them.
|
| We've used literal color names ("red", "blue", etc.) for the default
| palette, but if you'd rather use functional names like "primary" and
| "secondary", or even a numeric scale like "100" and "200", go for it.
|
*/

const colors = {
    transparent: 'transparent',

    'base-100': 'var(--base-100)',
    'base-200': 'var(--base-200)',
    'base-300': 'var(--base-300)',
    'base-400': 'var(--base-400)',
    'base-500': 'var(--base-500)',
    'base-600': 'var(--base-600)',
    'base-700': 'var(--base-700)',
    'base-800': 'var(--base-800)',
    'base-900': 'var(--base-900)',

    'primary-100': 'var(--primary-100)',
    'primary-200': 'var(--primary-200)',
    'primary-300': 'var(--primary-300)',
    'primary-400': 'var(--primary-400)',
    'primary-500': 'var(--primary-500)',
    'primary-600': 'var(--primary-600)',
    'primary-700': 'var(--primary-700)',
    'primary-800': 'var(--primary-800)',
    'primary-900': 'var(--primary-900)',

    'secondary-100': 'var(--secondary-100)',
    'secondary-200': 'var(--secondary-200)',
    'secondary-300': 'var(--secondary-300)',
    'secondary-400': 'var(--secondary-400)',
    'secondary-500': 'var(--secondary-500)',
    'secondary-600': 'var(--secondary-600)',
    'secondary-700': 'var(--secondary-700)',
    'secondary-800': 'var(--secondary-800)',
    'secondary-900': 'var(--secondary-900)',

    'tertiary-100': 'var(--tertiary-100)',
    'tertiary-200': 'var(--tertiary-200)',
    'tertiary-300': 'var(--tertiary-300)',
    'tertiary-400': 'var(--tertiary-400)',
    'tertiary-500': 'var(--tertiary-500)',
    'tertiary-600': 'var(--tertiary-600)',
    'tertiary-700': 'var(--tertiary-700)',
    'tertiary-800': 'var(--tertiary-800)',
    'tertiary-900': 'var(--tertiary-900)',

    'accent-100': 'var(--accent-100)',
    'accent-200': 'var(--accent-200)',
    'accent-300': 'var(--accent-300)',
    'accent-400': 'var(--accent-400)',
    'accent-500': 'var(--accent-500)',
    'accent-600': 'var(--accent-600)',
    'accent-700': 'var(--accent-700)',
    'accent-800': 'var(--accent-800)',
    'accent-900': 'var(--accent-900)',

    'success-100': 'var(--success-100)',
    'success-200': 'var(--success-200)',
    'success-300': 'var(--success-300)',
    'success-400': 'var(--success-400)',
    'success-500': 'var(--success-500)',
    'success-600': 'var(--success-600)',
    'success-700': 'var(--success-700)',
    'success-800': 'var(--success-800)',
    'success-900': 'var(--success-900)',

    'warning-100': 'var(--warning-100)',
    'warning-200': 'var(--warning-200)',
    'warning-300': 'var(--warning-300)',
    'warning-400': 'var(--warning-400)',
    'warning-500': 'var(--warning-500)',
    'warning-600': 'var(--warning-600)',
    'warning-700': 'var(--warning-700)',
    'warning-800': 'var(--warning-800)',
    'warning-900': 'var(--warning-900)',

    'caution-100': 'var(--caution-100)',
    'caution-200': 'var(--caution-200)',
    'caution-300': 'var(--caution-300)',
    'caution-400': 'var(--caution-400)',
    'caution-500': 'var(--caution-500)',
    'caution-600': 'var(--caution-600)',
    'caution-700': 'var(--caution-700)',
    'caution-800': 'var(--caution-800)',
    'caution-900': 'var(--caution-900)',

    'alert-100': 'var(--alert-100)',
    'alert-200': 'var(--alert-200)',
    'alert-300': 'var(--alert-300)',
    'alert-400': 'var(--alert-400)',
    'alert-500': 'var(--alert-500)',
    'alert-600': 'var(--alert-600)',
    'alert-700': 'var(--alert-700)',
    'alert-800': 'var(--alert-800)',
    'alert-900': 'var(--alert-900)'
};

module.exports = {
    /*
  |-----------------------------------------------------------------------------
  | Modules                  https://tailwindcss.com/docs/configuration#modules
  |-----------------------------------------------------------------------------
  |
  | Here is where you control which modules are generated and what variants are
  | generated for each of those modules.
  |
  | Currently supported variants: 'responsive',  'first-child', 'last-child', 'before', 'after', 'hover', 'focus'
  |
  | To disable a module completely, use `false` instead of an array.
  |
  */

    modules: {
        appearance: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundAttachment: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundColors: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        backgroundPosition: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundRepeat: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundSize: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderColors: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        borderRadius: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderStyle: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderWidths: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        cursor: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        display: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        flexbox: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        float: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        fonts: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        fontWeights: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        height: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        leading: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        lists: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        margin: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        maxHeight: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        maxWidth: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        minHeight: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        minWidth: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        negativeMargin: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        opacity: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        overflow: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        padding: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        pointerEvents: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        position: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        resize: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        shadows: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        svgFill: [],
        svgStroke: [],
        textAlign: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        textColors: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        textSizes: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        textStyle: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover'],
        tracking: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        userSelect: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        verticalAlign: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        visibility: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        whitespace: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        width: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        zIndex: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover']
    },

    /*
  |-----------------------------------------------------------------------------
  | Advanced Options         https://tailwindcss.com/docs/configuration#options
  |-----------------------------------------------------------------------------
  |
  | Here is where you can tweak advanced configuration options. We recommend
  | leaving these options alone unless you absolutely need to change them.
  |
  */

    options: {
        prefix: '',
        important: true,
        separator: ':'
    },
    /*
  |-----------------------------------------------------------------------------
  | Colors                                  https://tailwindcss.com/docs/colors
  |-----------------------------------------------------------------------------
  |
  | The color palette defined above is also assigned to the "colors" key of
  | your Tailwind config. This makes it easy to access them in your CSS
  | using Tailwind's config helper. For example:
  |
  | .error { color: config('colors.red') }
  |
  */

    colors,

    /*
  |-----------------------------------------------------------------------------
  | Screens                      https://tailwindcss.com/docs/responsive-design
  |-----------------------------------------------------------------------------
  |
  | Screens in Tailwind are translated to CSS media queries. They define the
  | responsive breakpoints for your project. By default Tailwind takes a
  | "mobile first" approach, where each screen size represents a minimum
  | viewport width. Feel free to have as few or as many screens as you
  | want, naming them in whatever way you'd prefer for your project.
  |
  | Tailwind also allows for more complex screen definitions, which can be
  | useful in certain situations. Be sure to see the full responsive
  | documentation for a complete list of options.
  |
  | Class name: .{screen}:{utility}
  |
  */

    screens: {
        sm: remCalc('576'),
        md: remCalc('768'),
        lg: remCalc('992'),
        xl: remCalc('1440'),
        xxl: remCalc('1830'),
        xxxl: remCalc('2130')
    },

    /*
  |-----------------------------------------------------------------------------
  | Fonts                                    https://tailwindcss.com/docs/fonts
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your project's font stack, or font families.
  | Keep in mind that Tailwind doesn't actually load any fonts for you.
  | If you're using custom fonts you'll need to import them prior to
  | defining them here.
  |
  | By default we provide a native font stack that works remarkably well on
  | any device or OS you're using, since it just uses the default fonts
  | provided by the platform.
  |
  | Class name: .font-{name}
  |
  */

    fonts: {
        condensed: ['Open Sans Condensed', 'sans-serif'],
        sans: [
            'Open Sans',
            '-apple-system',
            'BlinkMacSystemFont',
            'Segoe UI',
            'Roboto',
            'Oxygen',
            'Ubuntu',
            'Cantarell',
            'Fira Sans',
            'Droid Sans',
            'Helvetica Neue',
            'sans-serif'
        ],
        serif: [
            'Constantia',
            'Lucida Bright',
            'Lucidabright',
            'Lucida Serif',
            'Lucida',
            'DejaVu Serif',
            'Bitstream Vera Serif',
            'Liberation Serif',
            'Georgia',
            'serif'
        ],
        mono: ['Menlo', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', 'monospace']
    },

    /*
  |-----------------------------------------------------------------------------
  | Text sizes                         https://tailwindcss.com/docs/text-sizing
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your text sizes. Name these in whatever way
  | makes the most sense to you. We use size names by default, but
  | you're welcome to use a numeric scale or even something else
  | entirely.
  |
  | By default Tailwind uses the "rem" unit type for most measurements.
  | This allows you to set a root font size which all other sizes are
  | then based on. That said, you are free to use whatever units you
  | prefer, be it rems, ems, pixels or other.
  |
  | Class name: .text-{size}
  |
  */

    textSizes: {
        '2xs': remCalc('9'),
        xs: remCalc('10'),
        sm: remCalc('11'),
        base: remCalc('12'),
        lg: remCalc('13'),
        xl: remCalc('14'),
        '2xl': remCalc('16'),
        '3xl': remCalc('20'),
        '4xl': remCalc('24'),
        '5xl': remCalc('30'),
        '6xl': remCalc('40')
    },

    /*
  |-----------------------------------------------------------------------------
  | Font weights                       https://tailwindcss.com/docs/font-weight
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your font weights. We've provided a list of
  | common font weight names with their respective numeric scale values
  | to get you started. It's unlikely that your project will require
  | all of these, so we recommend removing those you don't need.
  |
  | Class name: .font-{weight}
  |
  */

    fontWeights: {
        '100': 100,
        '200': 200,
        '300': 300,
        '400': 400,
        '500': 500,
        '600': 600,
        '700': 700,
        '800': 800,
        '900': 900
    },

    /*
  |-----------------------------------------------------------------------------
  | Leading (line height)              https://tailwindcss.com/docs/line-height
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your line height values, or as we call
  | them in Tailwind, leadings.
  |
  | Class name: .leading-{size}
  |
  */

    leading: {
        none: 1,
        tight: 1.25,
        normal: 1.5,
        loose: 2
    },

    /*
  |-----------------------------------------------------------------------------
  | Tracking (letter spacing)       https://tailwindcss.com/docs/letter-spacing
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your letter spacing values, or as we call
  | them in Tailwind, tracking.
  |
  | Class name: .tracking-{size}
  |
  */

    tracking: {
        tight: `-${remCalc('1')}`,
        normal: '0',
        wide: remCalc('.5'),
        widest: remCalc('1')
    },

    /*
  |-----------------------------------------------------------------------------
  | Text colors                         https://tailwindcss.com/docs/text-color
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your text colors. By default these use the
  | color palette we defined above, however you're welcome to set these
  | independently if that makes sense for your project.
  |
  | Class name: .text-{color}
  |
  */

    textColors: colors,

    /*
  |-----------------------------------------------------------------------------
  | Background colors             https://tailwindcss.com/docs/background-color
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your background colors. By default these use
  | the color palette we defined above, however you're welcome to set
  | these independently if that makes sense for your project.
  |
  | Class name: .bg-{color}
  |
  */

    backgroundColors: colors,

    /*
  |-----------------------------------------------------------------------------
  | Border widths                     https://tailwindcss.com/docs/border-width
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your border widths. Take note that border
  | widths require a special "default" value set as well. This is the
  | width that will be used when you do not specify a border width.
  |
  | Class name: .border{-side?}{-width?}
  |
  */

    borderWidths: {
        default: '1px',
        '0': '0',
        '2': '2px',
        '3': '3px',
        '4': '4px',
        '8': '8px'
    },

    /*
  |-----------------------------------------------------------------------------
  | Border colors                     https://tailwindcss.com/docs/border-color
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your border colors. By default these use the
  | color palette we defined above, however you're welcome to set these
  | independently if that makes sense for your project.
  |
  | Take note that border colors require a special "default" value set
  | as well. This is the color that will be used when you do not
  | specify a border color.
  |
  | Class name: .border-{color}
  |
  */

    borderColors: Object.assign(
        {
            default: colors['grey-light']
        },
        colors
    ),

    /*
  |-----------------------------------------------------------------------------
  | Border radius                    https://tailwindcss.com/docs/border-radius
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your border radius values. If a `default` radius
  | is provided, it will be made available as the non-suffixed `.rounded`
  | utility.
  |
  | If your scale includes a `0` value to reset already rounded corners, it's
  | a good idea to put it first so other values are able to override it.
  |
  | Class name: .rounded{-side?}{-size?}
  |
  */

    borderRadius: {
        none: '0',
        sm: remCalc('2'),
        default: remCalc('4'),
        lg: remCalc('8'),
        full: '9999px'
    },

    /*
  |-----------------------------------------------------------------------------
  | Width                                    https://tailwindcss.com/docs/width
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your width utility sizes. These can be
  | percentage based, pixels, rems, or any other units. By default
  | we provide a sensible rem based numeric scale, a percentage
  | based fraction scale, plus some other common use-cases. You
  | can, of course, modify these values as needed.
  |
  |
  | It's also worth mentioning that Tailwind automatically escapes
  | invalid CSS class name characters, which allows you to have
  | awesome classes like .w-2/3.
  |
  | Class name: .w-{size}
  |
  */

    width: {
        auto: 'auto',
        px: '1px',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem',
        '10': '2.5rem',
        '12': '3rem',
        '16': '4rem',
        '18': '4.25rem',
        '20': '4.5rem',
        '24': '6rem',
        '32': '8rem',
        '43': '10.875rem',
        '48': '12rem',
        '55': '13.75rem',
        '64': '16rem',
        '1/2': '50%',
        '1/3': '33.33333%',
        '2/3': '66.66667%',
        '1/4': '25%',
        '3/4': '75%',
        '1/5': '20%',
        '2/5': '40%',
        '3/5': '60%',
        '4/5': '80%',
        '1/6': '16.66667%',
        '1/8': '12.5%',
        '1/10': '10%',
        '5/6': '83.33333%',
        full: '100%',
        screen: '100vw'
    },

    /*
  |-----------------------------------------------------------------------------
  | Height                                  https://tailwindcss.com/docs/height
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your height utility sizes. These can be
  | percentage based, pixels, rems, or any other units. By default
  | we provide a sensible rem based numeric scale plus some other
  | common use-cases. You can, of course, modify these values as
  | needed.
  |
  | Class name: .h-{size}
  |
  */

    height: {
        auto: 'auto',
        px: '1px',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem',
        '9': '2.25rem',
        '10': '2.5rem',
        '12': '3rem',
        '14': '3.5rem',
        '16': '4rem',
        '18': '4.25rem',
        '20': '4.5rem',
        '24': '6rem',
        '32': '8rem',
        '43': '10.875rem',
        '48': '12rem',
        '55': '13.75rem',
        '64': '16rem',
        '72': '20rem',
        full: '100%',
        screen: '100vh'
    },

    /*
  |-----------------------------------------------------------------------------
  | Minimum width                        https://tailwindcss.com/docs/min-width
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your minimum width utility sizes. These can
  | be percentage based, pixels, rems, or any other units. We provide a
  | couple common use-cases by default. You can, of course, modify
  | these values as needed.
  |
  | Class name: .min-w-{size}
  |
  */

    minWidth: {
        '0': '0',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem',
        '10': '2.5rem',
        '12': '3rem',
        '16': '4rem',
        '18': '4.25rem',
        '20': '4.5rem',
        '24': '6rem',
        '32': '8rem',
        '43': '10.875rem',
        '48': '12rem',
        '55': '13.75rem',
        '64': '16rem',
        '72': '18rem',
        '108': '26.875rem',
        '1/2': '50%',
        '1/3': '33.33333%',
        '2/3': '66.66667%',
        '1/4': '25%',
        '3/4': '75%',
        '1/5': '20%',
        '2/5': '40%',
        '3/5': '60%',
        '4/5': '80%',
        '1/6': '16.66667%',
        '1/8': '12.5%',
        '5/6': '83.33333%',
        full: '100%',
        fit: 'fit-content',
        min: 'min-content',
        max: 'max-content'
    },

    /*
  |-----------------------------------------------------------------------------
  | Minimum height                      https://tailwindcss.com/docs/min-height
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your minimum height utility sizes. These can
  | be percentage based, pixels, rems, or any other units. We provide a
  | few common use-cases by default. You can, of course, modify these
  | values as needed.
  |
  | Class name: .min-h-{size}
  |
  */

    minHeight: {
        '0': '0',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem',
        '9': '2.25rem',
        '10': '2.5rem',
        '12': '3rem',
        '14': '3.25rem',
        '16': '4rem',
        '18': '4.25rem',
        '20': '4.5rem',
        '24': '6rem',
        '32': '8rem',
        '43': '10.875rem',
        '48': '12rem',
        '55': '13.75rem',
        '64': '16rem',
        '1/2': '50%',
        '1/3': '33.33333%',
        '2/3': '66.66667%',
        '1/4': '25%',
        '3/4': '75%',
        '1/5': '20%',
        '2/5': '40%',
        '3/5': '60%',
        '4/5': '80%',
        '1/6': '16.66667%',
        '1/8': '12.5%',
        '5/6': '83.33333%',
        full: '100%',
        screen: '100vh'
    },

    /*
  |-----------------------------------------------------------------------------
  | Maximum width                        https://tailwindcss.com/docs/max-width
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your maximum width utility sizes. These can
  | be percentage based, pixels, rems, or any other units. By default
  | we provide a sensible rem based scale and a "full width" size,
  | which is basically a reset utility. You can, of course,
  | modify these values as needed.
  |
  | Class name: .max-w-{size}
  |
  */

    maxWidth: {
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem',
        '10': '2.5rem',
        '12': '3rem',
        '16': '4rem',
        '18': '4.25rem',
        '20': '4.5rem',
        '24': '6rem',
        '32': '8rem',
        '43': '10.875rem',
        '48': '12rem',
        '55': '13.75rem',
        '64': '16rem',
        '1/2': '50%',
        '1/3': '33.33333%',
        '2/3': '66.66667%',
        '1/4': '25%',
        '3/4': '75%',
        '1/5': '20%',
        '2/5': '40%',
        '3/5': '60%',
        '4/5': '80%',
        '1/6': '16.66667%',
        '1/8': '12.5%',
        '5/6': '83.33333%',
        xs: '20rem',
        sm: '30rem',
        md: '40rem',
        lg: '50rem',
        xl: '60rem',
        '2xl': '70rem',
        '3xl': '80rem',
        '4xl': '90rem',
        '5xl': '100rem',
        full: '100%',
        fit: 'fit-content',
        min: 'min-content',
        max: 'max-content'
    },

    /*
  |-----------------------------------------------------------------------------
  | Maximum height                      https://tailwindcss.com/docs/max-height
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your maximum height utility sizes. These can
  | be percentage based, pixels, rems, or any other units. We provide a
  | couple common use-cases by default. You can, of course, modify
  | these values as needed.
  |
  | Class name: .max-h-{size}
  |
  */

    maxHeight: {
        full: '100%',
        screen: '100vh'
    },

    /*
  |-----------------------------------------------------------------------------
  | Padding                                https://tailwindcss.com/docs/padding
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your padding utility sizes. These can be
  | percentage based, pixels, rems, or any other units. By default we
  | provide a sensible rem based numeric scale plus a couple other
  | common use-cases like "1px". You can, of course, modify these
  | values as needed.
  |
  | Class name: .p{side?}-{size}
  |
  */

    padding: {
        px: '1px',
        '0': '0',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem'
    },

    /*
  |-----------------------------------------------------------------------------
  | Margin                                  https://tailwindcss.com/docs/margin
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your margin utility sizes. These can be
  | percentage based, pixels, rems, or any other units. By default we
  | provide a sensible rem based numeric scale plus a couple other
  | common use-cases like "1px". You can, of course, modify these
  | values as needed.
  |
  | Class name: .m{side?}-{size}
  |
  */

    margin: {
        auto: 'auto',
        px: '1px',
        '0': '0',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '5': '1.25rem',
        '6': '1.5rem',
        '7': '1.75rem',
        '8': '2rem'
    },

    /*
  |-----------------------------------------------------------------------------
  | Negative margin                https://tailwindcss.com/docs/negative-margin
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your negative margin utility sizes. These can
  | be percentage based, pixels, rems, or any other units. By default we
  | provide matching values to the padding scale since these utilities
  | generally get used together. You can, of course, modify these
  | values as needed.
  |
  | Class name: .-m{side?}-{size}
  |
  */

    negativeMargin: {
        px: '1px',
        '0': '0',
        '1': '0.25rem',
        '2': '0.5rem',
        '3': '0.75rem',
        '4': '1rem',
        '6': '1.5rem',
        '8': '2rem'
    },

    /*
  |-----------------------------------------------------------------------------
  | Shadows                                https://tailwindcss.com/docs/shadows
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your shadow utilities. As you can see from
  | the defaults we provide, it's possible to apply multiple shadows
  | per utility using comma separation.
  |
  | If a `default` shadow is provided, it will be made available as the non-
  | suffixed `.shadow` utility.
  |
  | Class name: .shadow-{size?}
  |
  */

    shadows: {
        default: '0 2px 8px 0 hsla(0, 0%, 0%, 0.14)',
        md: '0 8px 8px 0 hsla(0, 0%, 0%, 0.04), 0 2px 4px 0 hsla(0, 0%, 0%, 0.17)',
        lg: '0 8px 8px 0 hsla(0, 0%, 0%, 0.04), 0 2px 4px 0 hsla(0, 0%, 0%, 0.17)',
        inner: 'inset 0 0px 8px 0 hsla(0, 0%, 0%, .25)',
        none: 'none'
    },

    /*
  |-----------------------------------------------------------------------------
  | Z-index                                https://tailwindcss.com/docs/z-index
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your z-index utility values. By default we
  | provide a sensible numeric scale. You can, of course, modify these
  | values as needed.
  |
  | Class name: .z-{index}
  |
  */

    zIndex: {
        auto: 'auto',
        '0': 0,
        '1': 1,
        '10': 10,
        '20': 20,
        '30': 30,
        '40': 40,
        '50': 50,
        '60': 60
    },

    /*
  |-----------------------------------------------------------------------------
  | Opacity                                https://tailwindcss.com/docs/opacity
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your opacity utility values. By default we
  | provide a sensible numeric scale. You can, of course, modify these
  | values as needed.
  |
  | Class name: .opacity-{name}
  |
  */

    opacity: {
        '0': '0',
        '25': '.25',
        '50': '.5',
        '75': '.75',
        '100': '1'
    },

    /*
  |-----------------------------------------------------------------------------
  | SVG fill                                   https://tailwindcss.com/docs/svg
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your SVG fill colors. By default we just provide
  | `fill-current` which sets the fill to the current text color. This lets you
  | specify a fill color using existing text color utilities and helps keep the
  | generated CSS file size down.
  |
  | Class name: .fill-{name}
  |
  */

    svgFill: {
        current: 'currentColor'
    },

    /*
  |-----------------------------------------------------------------------------
  | SVG stroke                                 https://tailwindcss.com/docs/svg
  |-----------------------------------------------------------------------------
  |
  | Here is where you define your SVG stroke colors. By default we just provide
  | `stroke-current` which sets the stroke to the current text color. This lets
  | you specify a stroke color using existing text color utilities and helps
  | keep the generated CSS file size down.
  |
  | Class name: .stroke-{name}
  |
  */

    svgStroke: {
        current: 'currentColor'
    },

    /*
  |-----------------------------------------------------------------------------
  | Modules                  https://tailwindcss.com/docs/configuration#modules
  |-----------------------------------------------------------------------------
  |
  | Here is where you control which modules are generated and what variants are
  | generated for each of those modules.
  |
  | Currently supported variants:
  |   - responsive
  |   - hover
  |   - focus
  |   - focus-within
  |   - active
  |   - group-hover
  |
  | To disable a module completely, use `false` instead of an array.
  |
  */

    modules: {
        appearance: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundAttachment: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundColors: [
            'responsive',
            'first-child',
            'last-child',
            'before',
            'after',
            'hover',
            'focus'
        ],
        backgroundPosition: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundRepeat: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        backgroundSize: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderCollapse: [],
        borderColors: [
            'responsive',
            'first-child',
            'last-child',
            'before',
            'after',
            'hover',
            'focus'
        ],
        borderRadius: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderStyle: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        borderWidths: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        cursor: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        display: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        flexbox: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        float: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        fonts: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        fontWeights: [
            'responsive',
            'first-child',
            'last-child',
            'before',
            'after',
            'hover',
            'focus'
        ],
        height: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        leading: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        lists: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        margin: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        maxHeight: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        maxWidth: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        minHeight: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        minWidth: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        negativeMargin: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        opacity: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        outline: ['focus'],
        overflow: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        padding: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        pointerEvents: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        position: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        resize: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        shadows: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover', 'focus'],
        svgFill: [],
        svgStroke: [],
        tableLayout: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        textAlign: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        textColors: [
            'responsive',
            'first-child',
            'last-child',
            'before',
            'after',
            'hover',
            'focus'
        ],
        textSizes: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        textStyle: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover', 'focus'],
        tracking: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        userSelect: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        verticalAlign: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        visibility: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        whitespace: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        width: ['responsive', 'first-child', 'last-child', 'before', 'after'],
        zIndex: ['responsive', 'hover', 'first-child', 'last-child', 'before', 'after']
    },

    /*
  |-----------------------------------------------------------------------------
  | Plugins                                https://tailwindcss.com/docs/plugins
  |-----------------------------------------------------------------------------
  |
  | Here is where you can register any plugins you'd like to use in your
  | project. Tailwind's built-in `container` plugin is enabled by default to
  | give you a Bootstrap-style responsive container component out of the box.
  |
  | Be sure to view the complete plugin documentation to learn more about how
  | the plugin system works.
  |
  */

    plugins: [
        function addvariant({ addVariant }) {
            addVariant('first-child', ({ modifySelectors, separator }) => {
                modifySelectors(({ className }) => `.fc${separator}${className} > *:first-child`);
            });
            addVariant('last-child', ({ modifySelectors, separator }) => {
                modifySelectors(({ className }) => `.lc${separator}${className} > *:last-child`);
            });
            addVariant('before', ({ modifySelectors, separator }) => {
                modifySelectors(({ className }) => `.before${separator}${className}:before`);
            });
            addVariant('after', ({ modifySelectors, separator }) => {
                modifySelectors(({ className }) => `.after${separator}${className}:after`);
            });
        },
        require('./tailwind-plugins/gradient.js')({
            variants: ['responsive', 'first-child', 'last-child', 'before', 'after', 'hover']
        }),

        require('./tailwind-plugins/object-fit.js')({
            variants: ['responsive', 'first-child', 'last-child', 'before', 'after']
        }),

        require('./tailwind-plugins/grid.js')({
            gaps: {
                '0': '0',
                '1px': '1px',
                '1': '0.25rem',
                '2': '0.5rem',
                '3': '0.75rem',
                '4': '1rem',
                '5': '1.25rem',
                '6': '1.5rem',
                '7': '1.75rem',
                '8': '2rem',
                '10': '2.5rem',
                '12': '3rem',
                '16': '4rem'
            },
            variants: ['responsive', 'first-child', 'last-child', 'before', 'after']
        }),

        require('./tailwind-plugins/order.js')({
            positive: {
                '0': 0,
                '1': 1,
                '2': 2,
                '3': 3,
                '4': 4,
                '5': 5
            },
            negative: {
                '-1': -1,
                '-2': -2,
                '-3': -3,
                '-4': -4,
                '-5': -5
            },
            variants: ['responsive', 'first-child', 'last-child', 'before', 'after']
        }),
        require('./tailwind-plugins/columns.js')({
            index: {
                '0': 0,
                '1': 1,
                '2': 2,
                '3': 3,
                '4': 4,
                '5': 5
            },
            variants: ['responsive', 'first-child', 'last-child', 'before', 'after']
        })
    ],

    /*
  |-----------------------------------------------------------------------------
  | Advanced Options         https://tailwindcss.com/docs/configuration#options
  |-----------------------------------------------------------------------------
  |
  | Here is where you can tweak advanced configuration options. We recommend
  | leaving these options alone unless you absolutely need to change them.
  |
  */

    options: {
        prefix: '',
        important: true,
        separator: ':'
    }
};
