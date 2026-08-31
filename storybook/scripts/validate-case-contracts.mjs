import fs from 'node:fs';
import path from 'node:path';

const storybook = path.resolve(process.argv[2] || 'storybook');
const domains = [
  'agent',
  'data-knowledge',
  'document-intelligence',
  'evaluation',
  'operations',
  'workflow',
];
const requiredHeadings = [
  '1. 用户目标与产品承诺',
  '2. 覆盖范围与边界',
  '3. 受控输入与金标',
  '4. 前置条件',
  '5. 执行路径',
  '6. 验收断言',
  '7. 证据、报告与失败口径',
  '8. 资源生命周期',
  '9. 变更影响与维护',
  '10. 公开边界',
];
const prohibited = [
  /<[^>]+>/,
  /\b(todo|tbd)\b/i,
  /https?:\/\//i,
  /github\.com/i,
  /issue\s*#/i,
  /#\d{3,}/,
  /s3:\/\//i,
  /matrixflow/i,
  /matrixorigin/i,
  /moi\.matrixorigin/i,
  /\b(api[_ -]?key|password|secret)\b/i,
];

function sectionBody(content, heading, nextHeading) {
  const start = content.indexOf(`## ${heading}`);
  if (start === -1) return '';
  const begin = start + `## ${heading}`.length;
  const end = nextHeading ? content.indexOf(`## ${nextHeading}`, begin) : content.length;
  return content.slice(begin, end === -1 ? content.length : end).trim();
}

const errors = [];
let checked = 0;

for (const domain of domains) {
  const directory = path.join(storybook, domain);
  if (!fs.existsSync(directory)) {
    errors.push(`${domain}: missing domain directory`);
    continue;
  }

  for (const filename of fs.readdirSync(directory).sort()) {
    if (!filename.endsWith('.md') || filename === 'README.md') continue;
    checked += 1;
    const file = path.join(directory, filename);
    const rel = path.relative(storybook, file);
    const content = fs.readFileSync(file, 'utf8');
    const lines = content.trimEnd().split('\n');
    const tables = lines.filter((line) => /^\|---/.test(line)).length;

    if (!/^# SB-[A-Z]{2,3}-\d{3}/m.test(content)) errors.push(`${rel}: case identifier/title missing`);
    if (!content.includes('| 门禁级别 | `required` |')) errors.push(`${rel}: required gate missing`);
    if (!content.includes('| 当前运行结果 | `not run` |')) errors.push(`${rel}: initial run status must be not run`);
    if (!content.includes('| 夹具管理 |')) errors.push(`${rel}: fixture management metadata missing`);
    if (lines.length < 120) errors.push(`${rel}: ${lines.length} lines; minimum is 120`);
    if (tables < 5) errors.push(`${rel}: ${tables} tables; minimum is 5`);
    if (!content.includes('### 负向路径')) errors.push(`${rel}: negative path missing`);
    if (!content.includes('### 人工验收边界')) errors.push(`${rel}: manual-acceptance boundary missing`);

    for (const heading of requiredHeadings) {
      if (!content.includes(`## ${heading}`)) errors.push(`${rel}: missing heading "${heading}"`);
    }
    for (let index = 0; index < requiredHeadings.length; index += 1) {
      const body = sectionBody(content, requiredHeadings[index], requiredHeadings[index + 1]);
      if (body.length < 80) errors.push(`${rel}: section ${index + 1} is too short to be executable`);
    }
    if (!/\| A\d+ \||^\d+\. |^- /m.test(sectionBody(content, requiredHeadings[5], requiredHeadings[6]))) {
      errors.push(`${rel}: no case-specific acceptance assertion found`);
    }
    if (!sectionBody(content, requiredHeadings[9]).includes('不公开')) {
      errors.push(`${rel}: public boundary must state what is not disclosed`);
    }
    for (const pattern of prohibited) {
      if (pattern.test(content)) errors.push(`${rel}: prohibited public-content pattern ${pattern}`);
    }
  }
}

if (checked === 0) errors.push('no Case documents found');
if (errors.length > 0) {
  console.error(`Storybook contract validation failed (${errors.length} issue(s)):`);
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log(`Storybook contract validation passed: ${checked} Case documents checked.`);
