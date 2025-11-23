import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration matching requirements:
// - Up to 20 teams and 200 users
// - RPS: 5
// - Response time SLI: 300ms
// - Success rate SLI: 99.9%
export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 5, // 5 RPS as per requirements
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<300'], // 95% under 300ms (SLI requirement)
    http_req_failed: ['rate<0.001'],   // Success rate > 99.9%
    errors: ['rate<0.001'],            // Error rate < 0.1%
  },
};

const BASE_URL = 'http://localhost:8080';

// Pre-created test data to avoid conflicts
const TEAMS = [];
const USERS = [];
const PRS = [];

export function setup() {
  console.log('Setting up test data: 20 teams, 200 users...');
  
  // Create 20 teams with 10 users each = 200 users total
  for (let teamIdx = 0; teamIdx < 20; teamIdx++) {
    const teamName = `team-${teamIdx}`;
    const members = [];
    
    for (let userIdx = 0; userIdx < 10; userIdx++) {
      const userId = `u-t${teamIdx}-${userIdx}`;
      members.push({
        user_id: userId,
        username: `user_${teamIdx}_${userIdx}`,
        is_active: true
      });
      USERS.push(userId);
    }
    
    const teamPayload = {
      team_name: teamName,
      members: members
    };
    
    const res = http.post(
      `${BASE_URL}/team/add`,
      JSON.stringify(teamPayload),
      { headers: { 'Content-Type': 'application/json' } }
    );
    
    if (res.status === 201 || res.status === 200) {
      TEAMS.push({ name: teamName, members: members });
      console.log(`Created team ${teamName} with ${members.length} users`);
    } else {
      console.error(`Failed to create team ${teamName}: ${res.status}`);
    }
  }
  
  console.log(`Setup complete: ${TEAMS.length} teams, ${USERS.length} users`);
  return { teams: TEAMS };
}

let prCounter = 0;

export default function (data) {
  if (!data.teams || data.teams.length === 0) {
    console.error('No teams available');
    return;
  }
  
  // Realistic workflow: Create PR -> Assign -> Query -> Merge -> Team management
  const scenario = Math.floor(Math.random() * 100);
  
  if (scenario < 35) {
    // 35% - Query operations (read-heavy)
    testGetStats();
  } else if (scenario < 60) {
    // 25% - Check user reviews
    testGetUserReviews(data);
  } else if (scenario < 75) {
    // 15% - Create new PR (with auto-assignment)
    testCreatePR(data);
  } else if (scenario < 88) {
    // 13% - Merge PR
    testMergePR(data);
  } else {
    // 12% - Deactivate team
    testDeactivateTeam(data);
  }
}

function testGetStats() {
  const res = http.get(`${BASE_URL}/stats`);
  
  const success = check(res, {
    'stats: status is 200': (r) => r.status === 200,
    'stats: response time < 300ms': (r) => r.timings.duration < 300,
    'stats: has total_teams': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.total_teams !== undefined;
      } catch {
        return false;
      }
    }
  });
  
  errorRate.add(!success);
}

function testCreatePR(data) {
  const team = data.teams[Math.floor(Math.random() * data.teams.length)];
  const author = team.members[0]; // First member as author
  
  // Generate unique PR ID and name using timestamp and random string
  const timestamp = Date.now();
  const random = randomString(8);
  const prId = `pr-${__VU}-${timestamp}-${random}`;
  
  const payload = {
    pull_request_id: prId,
    pull_request_name: `Feature-${timestamp}-${random}`,
    author_id: author.user_id
  };
  
  const res = http.post(
    `${BASE_URL}/pullRequest/create`,
    JSON.stringify(payload),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  const success = check(res, {
    'pr create: status is 201': (r) => r.status === 201,
    'pr create: response time < 300ms': (r) => r.timings.duration < 300,
  });
  
  errorRate.add(!success);
  
  // Store PR for later operations
  if (res.status === 201) {
    PRS.push(prId);
  }
}

function testAssignReviewers(data) {
  // Note: PRs are automatically assigned 2 reviewers on creation
  // This endpoint is for reassignment, which isn't tested in this load test
  // Skipping to avoid false 409 errors
  return;
}

function testMergePR(data) {
  if (PRS.length < 5) {
    // Need at least a few PRs before merging
    return;
  }
  
  const prId = PRS[Math.floor(Math.random() * Math.min(PRS.length, 10))]; // Merge recent PRs
  
  const payload = {
    pull_request_id: prId
  };
  
  const res = http.post(
    `${BASE_URL}/pullRequest/merge`,
    JSON.stringify(payload),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  const success = check(res, {
    'merge: status is 200': (r) => r.status === 200,
    'merge: response time < 300ms': (r) => r.timings.duration < 300,
  });
  
  errorRate.add(!success);
}

function testGetUserReviews(data) {
  const team = data.teams[Math.floor(Math.random() * data.teams.length)];
  const user = team.members[Math.floor(Math.random() * team.members.length)];
  
  const res = http.get(`${BASE_URL}/users/getReview?user_id=${user.user_id}`);
  
  const success = check(res, {
    'reviews: status is 200': (r) => r.status === 200,
    'reviews: response time < 300ms': (r) => r.timings.duration < 300,
  });
  
  errorRate.add(!success);
}

function testDeactivateTeam(data) {
  if (!data.teams || data.teams.length === 0) {
    return;
  }
  
  // Pick a random team to deactivate
  const team = data.teams[Math.floor(Math.random() * data.teams.length)];
  
  const payload = {
    team_name: team.name
  };
  
  const res = http.post(
    `${BASE_URL}/team/deactivate`,
    JSON.stringify(payload),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  const success = check(res, {
    'deactivate: status is 200 or 404': (r) => r.status === 200 || r.status === 404, // 404 if already deactivated or not found
    'deactivate: response time < 300ms': (r) => r.timings.duration < 300,
  });
  
  // Don't count 404 as error - team might be already deactivated
  errorRate.add(!success && res.status !== 404);
}

export function teardown(data) {
  console.log('Requirements validation test completed');
  console.log(`Teams created: ${data.teams.length}`);
  console.log(`PRs created during test: ${PRS.length}`);
}
