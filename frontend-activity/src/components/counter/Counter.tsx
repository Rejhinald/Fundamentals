import { useState } from 'react';
import {Button, Card} from 'react-bootstrap';
interface CounterProps {
  initialCount?: number;
  step?: number;
  onCountChange: (newCount: number) => void;
  cardStyle?: React.CSSProperties;
  buttonLabels?: {
    increment: string;
    decrement: string;
  };
}

export default function Counter({
  initialCount = 0, 
  step = 1, 
  onCountChange, 
  cardStyle = {}, 
  buttonLabels = { increment: "+1", decrement: "-1" } 
}: CounterProps) {
  const [count, setCount] = useState(initialCount);

  const addCount = () => {
    const newCount = count + step;
    setCount(newCount);
    onCountChange?.(newCount);
  };

  const minusCount = () => {
    const newCount = count - step;
    setCount(newCount);
    onCountChange?.(newCount);
  };

  return (
    <Card style={{ width: '14rem', ...cardStyle }}>
      <Card.Body>
        <p>Counter: {count}</p>
        <div className="d-flex justify-content-center gap-2">
          <Button
            onClick={minusCount}
            variant="danger"
            size="sm"
          >
            {buttonLabels.decrement}
          </Button>
          <Button
            onClick={addCount}
            variant="primary"
            size="sm"
          >
            {buttonLabels.increment}
          </Button>
        </div>
      </Card.Body>
    </Card>
  );
}
