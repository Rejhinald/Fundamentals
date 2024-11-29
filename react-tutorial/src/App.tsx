import ListGroup from "./components/ListGroup";
import Alert from "./components/Alert";
import Button from "./components/Button";
import { useState } from "react";

function App() {
  const [alertVisible, setalertVisible] = useState(false);





  const items = [
    "An item",
    "A second item",
    "A third item",
    "A fourth item",
    "And a fifth one",
  ];
  const items2 = ["New York", "Los Angeles", "Chicago", "Houston", "Phoenix"];
  const handleSelectItem = (item: string) => {
    console.log(item);
  }

  return (
    <div>
      <ListGroup items={items} heading="Items" onSelectItem={handleSelectItem} />
      <ListGroup items={items2} heading="Cities" onSelectItem={handleSelectItem} />
      { alertVisible && <Alert onClose={() => setalertVisible(false)}>
        <span>Hello 2</span> Hello World
        </Alert>}
        <Button color="success" onClick={() => setalertVisible(true)}>
          Click Me
        </Button>
    </div>
  );
}

export default App;
