import React from 'react';

const CSSGrid = () => (
    <div className="overflow-auto p-4 h-full w-full">
        <div className="bg-gradient-horizontal p-4 mb-4 text-base-100">hello world</div>
        <div className="grid grid-gap-4 md:grid-auto-fill md:grid-dense uppercase text-base-100 text-sm font-700">
            <div className="bg-base-700 p-4 rounded">item 1</div>
            <div className="bg-base-700 p-4 rounded">item 2</div>
            <div className="bg-base-700 p-4 rounded md:s-3">item 3</div>
            <div className="bg-base-700 p-4 rounded">item 4</div>
            <div className="grid grid-title md:s-2 md:grid-auto-fill md:grid-dense grid-gap-1px">
                <div className="bg-base-800 rounded-t p-4">
                    with optional section title (the first sibling)
                </div>
                <div className="bg-base-800 p-4 rounded">item 6</div>
                <div className="bg-base-800 p-4 rounded">item 7</div>
                <div className="bg-base-800 p-4 rounded">item 8</div>
                <div className="bg-base-800 p-4 rounded">item 9</div>
            </div>
            <div className="bg-base-700 p-4 rounded md:s-2">item 10</div>
            <div className="bg-base-700 p-4 rounded md:sx-2">item 11</div>
            <div className="bg-base-700 p-4 rounded">item 12</div>
            <div className="bg-base-700 p-4 rounded md:s-2">item 13</div>
            <div className="bg-base-700 p-4 rounded">item 14</div>
            <div className="bg-base-700 p-4 rounded">item 15</div>
            <div className="bg-base-700 p-4 rounded">item 16</div>
            <div className="bg-base-700 p-4 rounded md:s-3">item 17</div>
            <div className="bg-base-700 p-4 rounded">item 18</div>
            <div className="bg-base-700 p-4 rounded">item 19</div>
            <div className="bg-base-700 p-4 rounded">item 20</div>
            <div className="bg-base-700 p-4 rounded md:s-2">item 21</div>
            <div className="bg-base-700 p-4 rounded">item 22</div>
            <div className="bg-base-700 p-4 rounded">item 23</div>
            <div className="bg-base-700 p-4 rounded">item 24</div>
            <div className="bg-base-700 p-4 rounded">item 25</div>
            <div className="bg-base-700 p-4 rounded md:sx-3">item 26</div>
            <div className="bg-base-700 p-4 rounded">item 27</div>
            <div className="bg-base-700 p-4 rounded">item 28</div>
            <div className="bg-base-700 p-4 rounded">item 29</div>
            <div className="bg-base-700 p-4 rounded">item 30</div>
            <div className="bg-base-700 p-4 rounded">item 31</div>
            <div className="bg-base-700 p-4 rounded">item 32</div>
        </div>
    </div>
);

export default CSSGrid;
