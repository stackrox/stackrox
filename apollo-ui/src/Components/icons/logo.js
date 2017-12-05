import React, { Component } from "react";

// Icon
class Logo extends Component {

  render() {
    return (
      <svg
        className={`logo ${ this.props.className }`}
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="24"
        viewBox="0 0 180 104"
        aria-labelledby="StackRox">
        <title id="title">StackRox</title>
        <g id="Layer_1-2" data-name="Layer 1"><path d="M179.14,101.08a1.33,1.33,0,0,0,.78-1.7l-21.51-62a1.29,1.29,0,0,0-1.68-.76L100.89,65.73a1.33,1.33,0,0,0-.78,1.7l13.53,36.94H172Z"/><path d="M90.47,64.15a1.33,1.33,0,0,1,.78-1.7l53.26-27.74L139,18.95a1.65,1.65,0,0,0-2.15-1L65.53,55.15a1.7,1.7,0,0,0-1,2.17l17.24,47H105.2Z"/><path d="M54.71,54a1.7,1.7,0,0,1,1-2.17l65.73-34.24L115.76,1.25a1.92,1.92,0,0,0-2.5-1.12L30.48,43.25a2,2,0,0,0-1.16,2.51l21.47,58.6H73.15Z"/><path d="M22.07,48.23l20.54,56.13-34,0a.66.66,0,0,1-.63-.45L0,80.61A.66.66,0,0,1,.1,80L20.88,48.11A.66.66,0,0,1,22.07,48.23Z"/></g>


      </svg>
    );
  }
}

export default Logo;