import { useState } from 'react';
import Button from 'react-bootstrap/Button';
import Card from 'react-bootstrap/Card';

export default function Counter() {
  const [count, setCount] = useState(0);

  return (
    <>
      <Card style={{ width: '14rem' }}>
        <Card.Body>
          <p>Counter: {count}</p>
          <div className="d-flex justify-content-center gap-2">
            <Button
              onClick={() => setCount((count) => count - 1)}
              variant="danger"
              size="sm"
            >
              -1
            </Button>
            <Button
              onClick={() => setCount((count) => count + 1)}
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
