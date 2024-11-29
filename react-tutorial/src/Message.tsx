// PascalCasing is used for the file name and the function name
function Message() {
    const name = 'Arwin Miclat';
    if (name)
        return <h1>Hello, {name}</h1>; // JSX: Javascript XML
    return <h1>Hello, Stranger</h1>;
}

export default Message;