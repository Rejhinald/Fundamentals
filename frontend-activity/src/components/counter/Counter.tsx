import { useState } from 'react';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';

export default function Counter() {
  const [count, setCount] = useState(0);
  
  const addCount = () => {
    setCount(count + 1);
  }

  const minusCount = () => {
    setCount(count - 1);
  }

  return (
    <>
      <Card style={{ width: '14rem' }}>
        <Card.Body>
          <p>Counter: {count}</p>
          <div className="d-flex justify-content-center gap-2">
            <Button
              onClick={() => minusCount()}
              variant="danger"
              size="sm"
            >
              -1
            </Button>
            <Button
              onClick={() => addCount()}
              variant="primary"
              size="sm"
            >
              +1
            </Button>
          </div>
        </Card.Body>
      </Card>
    </>
  );
}
