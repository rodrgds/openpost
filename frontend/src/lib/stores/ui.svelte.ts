import { type DateValue } from '@internationalized/date';

class UIState {
	isComposeOpen = $state(false);
	composeInitialDate = $state<DateValue | undefined>(undefined);
	isDayPostsOpen = $state(false);
	dayPostsDate = $state<DateValue | undefined>(undefined);
	refreshCounter = $state(0);
	promptText = $state<string | null>(null);

	openCompose(date?: DateValue) {
		this.composeInitialDate = date;
		this.isComposeOpen = true;
	}

	closeCompose() {
		this.isComposeOpen = false;
	}

	openDayPosts(date: DateValue) {
		this.dayPostsDate = date;
		this.isDayPostsOpen = true;
	}

	closeDayPosts() {
		this.isDayPostsOpen = false;
	}

	openComposeForDay(date: DateValue) {
		this.isDayPostsOpen = false;
		this.composeInitialDate = date;
		this.isComposeOpen = true;
	}

	setPrompt(text: string) {
		this.promptText = text;
	}

	clearPrompt() {
		this.promptText = null;
	}

	triggerRefresh() {
		this.refreshCounter++;
	}
}

export const ui = new UIState();
