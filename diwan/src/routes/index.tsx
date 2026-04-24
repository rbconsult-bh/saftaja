import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

export const Route = createFileRoute('/')({
  component: Index,
})

function Index() {
  const navigate = useNavigate();

  useEffect(() => {
    // TODO: check if user is logged in or not
    let isLoggedIn = false;
    if (isLoggedIn) {
      // TODO: go to /select
    } else {
      // TODO: go to /auth
    }
  }, [])


  return (
    <h1>YOU SHOULD NEVER EVER SEE THIS :D</h1>
  )
}
