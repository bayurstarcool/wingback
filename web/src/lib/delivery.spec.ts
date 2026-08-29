import { describe, it, expect } from 'vitest';
import { formatCountdown, formatDistance } from './delivery';

describe('formatCountdown', () => {
	it('returns "Sudah sampai" when arrival is in the past', () => {
		const now = new Date('2026-01-01T12:00:00Z');
		const arrivesAt = new Date('2026-01-01T11:00:00Z').toISOString();
		expect(formatCountdown(arrivesAt, now)).toBe('Sudah sampai');
	});

	it('formats minutes and seconds', () => {
		const now = new Date('2026-01-01T12:00:00Z');
		const arrivesAt = new Date('2026-01-01T12:05:30Z').toISOString();
		expect(formatCountdown(arrivesAt, now)).toBe('5m 30d');
	});

	it('formats hours and minutes', () => {
		const now = new Date('2026-01-01T12:00:00Z');
		const arrivesAt = new Date('2026-01-01T15:30:00Z').toISOString();
		expect(formatCountdown(arrivesAt, now)).toBe('3j 30m');
	});

	it('formats days, hours, and minutes', () => {
		const now = new Date('2026-01-01T12:00:00Z');
		const arrivesAt = new Date('2026-01-03T14:00:00Z').toISOString();
		expect(formatCountdown(arrivesAt, now)).toBe('2h 2j 0m');
	});
});

describe('formatDistance', () => {
	it('formats sub-km distances as meters', () => {
		expect(formatDistance(0.5)).toBe('500 m');
	});

	it('formats km distances with one decimal', () => {
		expect(formatDistance(650.34)).toBe('650.3 km');
	});
});
