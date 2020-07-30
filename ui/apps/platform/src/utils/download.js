export default function download(filename, text, fileType) {
    const element = document.createElement('a');
    element.setAttribute('href', `data:text/${fileType};charset=utf-8,${encodeURIComponent(text)}`);
    element.setAttribute('download', filename);

    element.style.display = 'none';
    document.body.appendChild(element);

    element.click();

    document.body.removeChild(element);
}
