<script lang="ts">
	import { outerWidth } from 'svelte/reactivity/window';
	import Zdog from 'zdog';

	let width = $derived(Math.min(outerWidth.current || 0, 400) - 40);
	let canvas = $state<HTMLCanvasElement>();

	$effect(() => {
		if (canvas) {
			const illo = new Zdog.Illustration({
				element: canvas
			});

			const baseGroup = new Zdog.Group({
				addTo: illo,
				rotate: {
					x: 4.25,
					y: 0.8
				}
			});

			const dish = new Zdog.Hemisphere({
				addTo: baseGroup,
				diameter: 140,
				stroke: false,
				color: '#d4300b',
				backface: '#fb5939'
			});

			const antenna = new Zdog.Shape({
				addTo: dish,
				path: [
					{ x: 40, y: 40, z: 35 },
					{ x: 0, z: -70 }
				],
				stroke: 5,
				color: '#d4300b'
			});

			new Zdog.Shape({
				addTo: dish,
				path: [
					{ x: -40, z: 50 },
					{ x: 0, z: -70 }
				],
				stroke: 5,
				color: '#d4300b'
			});

			new Zdog.Shape({
				addTo: dish,
				path: [
					{ x: 0, y: -40, z: 50 },
					{ x: 0, z: -70 }
				],
				stroke: 5,
				color: '#d4300b'
			});

			// Add two hemispheres for a antenna topper
			const topper = new Zdog.Hemisphere({
				addTo: antenna,
				diameter: 10,
				stroke: 5,
				color: '#d4300b',
				backface: '#fb5939',
				translate: { z: -70 }
			});

			topper.copy({
				rotate: { x: Zdog.TAU / 2 }
			});

			const rotation = Zdog.TAU / 2.67; // 135 degrees in radians
			const turnDuration = 5; // time in seconds for a turn of radar
			let turned = false;

			const animate = () => {
				if (!turned && illo.rotate.y < rotation) {
					illo.rotate.y += rotation / (turnDuration * 60); // assuming 60fps
				} else if (!turned && illo.rotate.y >= rotation) {
					illo.rotate.y = rotation; // snap to exact position
					turned = true;
				} else if (turned && illo.rotate.y <= rotation && illo.rotate.y >= 0) {
					illo.rotate.y -= rotation / (turnDuration * 60);
				} else if (turned && illo.rotate.y < 0) {
					illo.rotate.y = 0; // snap back to original position
					turned = false;
				}

				illo.updateRenderGraph();
				requestAnimationFrame(animate);
			};

			animate();
		}
	});
</script>

<canvas bind:this={canvas} {width} height="300"></canvas>
